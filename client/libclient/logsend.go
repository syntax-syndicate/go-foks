package libclient

import (
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"strings"

	"github.com/foks-proj/go-foks/lib/chains"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type ChunkedLog struct {
	Path   core.Path
	Blocks []rem.LogSendBlob
	Hash   proto.StdHash
	Len    int
	Data   []byte
}

func (c *ChunkedLog) compressIfNeeded() error {
	if strings.HasSuffix(c.Path.Base().String(), ".gz") {
		return nil // already compressed
	}
	dat, err := c.Path.ReadFile()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(dat)
	if err != nil {
		return err
	}
	err = gz.Close()
	if err != nil {
		return err
	}
	if buf.Len() >= len(dat) {
		// Compression didn't help, so we won't use it.
		return nil
	}
	c.Data = buf.Bytes()
	c.Path = core.Path(c.Path.String() + ".gz")

	return nil
}

func ChunkLog(
	m MetaContext,
	path core.Path,
) (
	*ChunkedLog,
	error,
) {
	dat, err := path.ReadFile()
	if err != nil {
		return nil, err
	}
	ret := &ChunkedLog{
		Path: path,
		Len:  len(dat),
		Data: dat,
	}
	err = ret.compressIfNeeded()
	if err != nil {
		return nil, err
	}
	// Might have been compressed, so we need to re-read the data.
	dat = ret.Data

	h := sha512.New512_256()
	_, err = h.Write(dat)
	if err != nil {
		return nil, err
	}
	tmp := h.Sum(nil)
	if len(tmp) != len(ret.Hash) {
		return nil, core.InternalError("hash len mismatch")
	}
	copy(ret.Hash[:], tmp)
	chunkSize := 4 * 1024 * 1024 // 4 MiB

	for i := 0; i < len(dat); i += chunkSize {
		end := min(i+chunkSize, len(dat))
		chunk := rem.LogSendBlob(dat[i:end])
		ret.Blocks = append(ret.Blocks, chunk)
	}
	return ret, nil
}

func ListLatestLogs(m MetaContext, n int) ([]core.Path, error) {
	if n <= 0 {
		return nil, core.InternalError("bad arguments to ListLatestLogs: n must be > 0")
	}
	nm, err := m.G().OutLogPath()
	if err != nil {
		return nil, err
	}
	var ret []core.Path
	ret = append(ret, nm)
	n--
	if n <= 0 {
		return ret, nil
	}

	logs, err := FindRotatedLogs(m, nm)
	if err != nil {
		return nil, err
	}
	for _, log := range logs {
		if n <= 0 {
			break
		}
		ret = append(ret, log.Path)
		n--
	}
	return ret, err

}

type logSender struct {
	logs []core.Path
	srv  *chains.Probe
	cli  rem.LogSendClient
	id   proto.LogSendID
}

func (l *logSender) run(m MetaContext) error {

	err := l.getServer(m)
	if err != nil {
		return err
	}
	err = l.initSession(m)
	if err != nil {
		return err
	}
	for i, log := range l.logs {
		err = l.sendLog(m, i, log)
		if err != nil {
			m.Errorw("logsend failed", "err", err, "log", log)
		}
	}
	return nil
}

func (l *logSender) sendLog(m MetaContext, idx int, log core.Path) error {
	clog, err := ChunkLog(m, log)
	if err != nil {
		return err
	}
	nm := clog.Path.Base()
	fid := rem.LogSendFileID(idx)
	m.Infow("logsend", "id", l.id, "log", nm, "fileid", fid, "nchunks", len(clog.Blocks), "hash", clog.Hash, "stage", "sending")

	err = l.cli.LogSendInitFile(m.Ctx(), rem.LogSendInitFileArg{
		Id:      l.id,
		FileID:  fid,
		Name:    nm.Export(),
		Len:     proto.Size(clog.Len),
		NBlocks: uint64(len(clog.Blocks)),
		Hash:    clog.Hash,
	})
	if err != nil {
		return err
	}
	m.Infow("logsend", "id", l.id, "fileid", fid, "name", nm, "stage", "sent")
	for i, block := range clog.Blocks {
		m.Debugw("logsend", "id", l.id, "fileid", fid, "blockno", i, "stage", "sending block")
		err = l.cli.LogSendUploadBlock(m.Ctx(), rem.LogSendUploadBlockArg{
			Id:      l.id,
			FileID:  fid,
			BlockNo: uint64(i),
			Block:   rem.LogSendBlob(block),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *logSender) initSession(m MetaContext) error {
	id, err := l.cli.LogSendInit(m.Ctx())
	if err != nil {
		return err
	}
	l.id = id
	hn, err := l.srv.HostnameWithOptionalPort()
	if err != nil {
		return err
	}
	m.Infow("logsend", "id", l.id, "server", hn)
	return nil

}

func LogSend(m MetaContext, logs []core.Path) (*lcl.LogSendRes, error) {
	m = m.WithLogTag("logsend")
	ls := &logSender{
		logs: logs,
	}
	err := ls.run(m)
	if err != nil {
		return nil, err
	}
	hn, err := ls.srv.HostnameWithOptionalPort()
	if err != nil {
		m.Warnw("logsend", "stage", "gethostname", "err", err)
		hn = "-"
	}
	res := &lcl.LogSendRes{
		Id:   ls.id,
		Host: proto.TCPAddr(hn),
	}
	return res, nil
}

func (l *logSender) getServer(m MetaContext) error {
	au := m.G().ActiveUser()

	// If active user, then let's connect to that server and then use a logged-in
	if au != nil {
		gcli, err := au.UserGCli(m)
		if err != nil {
			return err
		}
		l.cli = core.NewLogSendClient(gcli, m)
		l.srv = au.HomeServer()
		return nil
	}

	srv := m.G().GetLastServer()
	if srv == nil {
		return core.NoDefaultHostError{}
	}

	// Otherwise, we'll connect to the reg server at the last server, and we have something
	// to try
	rcli, err := srv.RegGCli(m)
	if err != nil {
		return err
	}
	l.cli = core.NewLogSendClient(rcli, m)
	l.srv = srv

	return nil
}
