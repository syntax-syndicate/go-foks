package shared

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lib"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
	"github.com/jackc/pgx/v5"
)

func LogSendInit(
	m MetaContext,
) (
	proto.LogSendID,
	error,
) {
	var zed proto.LogSendID
	id, err := proto.NewLogSendID()
	if err != nil {
		return zed, err
	}
	db, err := m.Db(DbTypeUsers)
	if err != nil {
		return zed, err
	}
	var uidBytes []byte
	if !m.UID().IsZero() {
		uidBytes = m.UID().ExportToDB()
	}
	defer db.Release()
	tag, err := db.Exec(
		m.Ctx(),
		`INSERT INTO log_send (short_host_id, id, uid, ctime) 
		VALUES ($1, $2, $3, NOW())`,
		m.ShortHostID().ExportToDB(),
		id.ExportToDB(),
		uidBytes,
	)
	if err != nil {
		return zed, err
	}
	if tag.RowsAffected() != 1 {
		return zed, core.InsertError("log_send")
	}
	return *id, nil
}

func LogSendInitFile(
	m MetaContext,
	arg rem.LogSendInitFileArg,
) error {

	return RetryTxUserDB(m, "LogSendInitFile", func(m MetaContext, tx pgx.Tx) error {
		var dummy int
		err := tx.QueryRow(
			m.Ctx(),
			"SELECT 1 FROM log_send WHERE short_host_id = $1 AND id = $2",
			m.ShortHostID().ExportToDB(),
			arg.Id.ExportToDB(),
		).Scan(&dummy)
		if errors.Is(err, pgx.ErrNoRows) {
			return core.NotFoundError("log_send")
		}
		if err != nil {
			return err
		}

		tag, err := tx.Exec(
			m.Ctx(),
			`INSERT INTO log_send_files 
			(short_host_id, ls_id, file_id, filename, len, nblocks, hsh, ctime)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
			m.ShortHostID().ExportToDB(),
			arg.Id.ExportToDB(),
			int(arg.FileID),
			arg.Name,
			int(arg.Len),
			int(arg.NBlocks),
			arg.Hash.ExportToDB(),
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return core.InsertError("log_send_files")
		}
		return nil
	})
}

func LogSendUploadBlock(
	m MetaContext,
	arg rem.LogSendUploadBlockArg,
) error {
	return RetryTxUserDB(m, "LogSendUploadBlock", func(m MetaContext, tx pgx.Tx) error {
		var nblocks int
		err := tx.QueryRow(
			m.Ctx(),
			`SELECT nblocks FROM log_send_files WHERE short_host_id = $1 AND ls_id=$2 AND file_id = $3`,
			m.ShortHostID().ExportToDB(),
			arg.Id.ExportToDB(),
			int(arg.FileID),
		).Scan(&nblocks)
		if errors.Is(err, pgx.ErrNoRows) {
			return core.NotFoundError("log_send_files")
		}
		if err != nil {
			return err
		}
		if int(arg.BlockNo) >= nblocks {
			return core.BadArgsError("block number out of range")
		}
		tag, err := tx.Exec(
			m.Ctx(),
			`INSERT INTO log_send_blocks 
			(short_host_id, ls_id, file_id, block_id, block)
			VALUES ($1, $2, $3, $4, $5)`,
			m.ShortHostID().ExportToDB(),
			arg.Id.ExportToDB(),
			int(arg.FileID),
			int(arg.BlockNo),
			arg.Block[:],
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return core.InsertError("log_send_blocks")
		}
		return nil
	})
}

type LogSendFile struct {
	Name         lib.LocalFSPath
	RawData      []byte
	ExpandedData []byte
}

type LogSendSet struct {
	HostID proto.HostID
	UID    *proto.UID
	ID     proto.LogSendID
	Files  []LogSendFile
}

type logSendFile struct {
	FileID       rem.LogSendFileID
	Name         string
	Len          int
	NBlocks      int
	Hash         proto.StdHash
	Data         []byte
	ExpandedData []byte
}

type logSetReassembler struct {
	id     proto.LogSendID
	hostid *core.HostID
	uid    *proto.UID
	files  []*logSendFile
}

func (l *logSetReassembler) run(m MetaContext) error {
	err := l.fetchMetadata(m)
	if err != nil {
		return err
	}
	err = l.fetchFiles(m)
	if err != nil {
		return err
	}
	for _, file := range l.files {
		err = l.fetchBlocks(m, file)
		if err != nil {
			return err
		}
		err = file.expand()
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *logSetReassembler) res() *LogSendSet {
	ret := LogSendSet{
		HostID: l.hostid.Id,
		UID:    l.uid,
		ID:     l.id,
	}
	for _, file := range l.files {
		ret.Files = append(ret.Files, LogSendFile{
			Name:         lib.LocalFSPath(file.Name),
			RawData:      file.Data,
			ExpandedData: file.ExpandedData,
		})
	}
	return &ret
}

func LogSendReassemble(
	m MetaContext,
	id proto.LogSendID,
) (
	*LogSendSet,
	error,
) {
	l := &logSetReassembler{id: id}
	err := l.run(m)
	if err != nil {
		return nil, err
	}
	return l.res(), nil
}

func (l *logSetReassembler) fetchBlocks(
	m MetaContext,
	file *logSendFile,
) error {
	db, err := m.Db(DbTypeUsers)
	if err != nil {
		return err
	}
	defer db.Release()

	rows, err := db.Query(
		m.Ctx(),
		`SELECT block_id, block
		FROM log_send_blocks 
		WHERE short_host_id = $1 AND ls_id = $2 AND file_id = $3 
		ORDER BY block_id`,
		l.hostid.Short.ExportToDB(),
		l.id.ExportToDB(),
		int(file.FileID),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	blocks := make([][]byte, file.NBlocks)
	blockCount := 0

	for rows.Next() {
		var blockID int
		var data []byte
		err := rows.Scan(&blockID, &data)
		if err != nil {
			return err
		}

		if blockID < 0 || blockID >= file.NBlocks {
			return core.NotFoundError("block ID out of range")
		}

		if blocks[blockID] != nil {
			return core.DuplicateError("duplicate block")
		}

		blocks[blockID] = data
		blockCount++
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if blockCount != file.NBlocks {
		return core.NotFoundError("missing blocks")
	}

	var result []byte
	for i, block := range blocks {
		if block == nil {
			return core.NotFoundError(fmt.Sprintf("missing block %d", i))
		}
		result = append(result, block...)
	}
	file.Data = result
	return nil
}

func (l *logSetReassembler) fetchMetadata(m MetaContext) error {
	var hid int
	var uidBytes []byte
	db, err := m.Db(DbTypeUsers)
	if err != nil {
		return err
	}
	defer db.Release()

	err = db.QueryRow(
		m.Ctx(),
		`SELECT short_host_id, uid
		FROM log_send
		WHERE id = $1`,
		l.id.ExportToDB(),
	).Scan(&hid, &uidBytes)
	if errors.Is(err, pgx.ErrNoRows) {
		return core.NotFoundError("log_send")
	}
	if err != nil {
		return err
	}
	chid, err := m.G().HostIDMap().LookupByShortID(m, core.ShortHostID(hid))
	if err != nil {
		return err
	}
	l.hostid = chid
	if len(uidBytes) == 0 {
		return nil
	}
	var uid proto.UID
	err = uid.ImportFromDB(uidBytes)
	if err != nil {
		return err
	}
	l.uid = &uid
	return nil
}

func (l *logSetReassembler) fetchFiles(m MetaContext) error {
	db, err := m.Db(DbTypeUsers)
	if err != nil {
		return err
	}
	defer db.Release()

	rows, err := db.Query(
		m.Ctx(),
		`SELECT file_id, filename, len, nblocks, hsh 
		FROM log_send_files 
		WHERE short_host_id = $1 AND ls_id = $2 
		ORDER BY file_id ASC`,
		l.hostid.Short.ExportToDB(),
		l.id.ExportToDB(),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var files []*logSendFile
	for rows.Next() {
		var file logSendFile
		var hashBytes []byte
		var fileID int
		err := rows.Scan(&fileID, &file.Name, &file.Len, &file.NBlocks, &hashBytes)
		if err != nil {
			return err
		}
		file.FileID = rem.LogSendFileID(fileID)
		if len(hashBytes) != len(file.Hash) {
			return core.BadServerDataError("hash length mismatch")
		}
		copy(file.Hash[:], hashBytes)
		files = append(files, &file)
	}

	if err := rows.Err(); err != nil {
		return err
	}
	if len(files) == 0 {
		return core.NotFoundError("no files in log send")
	}
	l.files = files
	return nil
}

func (f *logSendFile) expand() error {
	if !strings.HasSuffix(f.Name, ".gz") {
		return nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(f.Data))
	if err != nil {
		return err
	}
	defer reader.Close()

	expanded, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	f.ExpandedData = expanded
	return nil
}
