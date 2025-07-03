// Auto-generated to Go types and interfaces using snowpc 0.0.4 (https://github.com/foks-proj/go-snowpack-compiler)
//  Input file:../../proto-src/rem/logsend.snowp

package rem

import (
	"context"
	"errors"
	"github.com/foks-proj/go-snowpack-rpc/rpc"
	"time"
)

import lib "github.com/foks-proj/go-foks/proto/lib"

type LogSendFileID uint64
type LogSendFileIDInternal__ uint64

func (l LogSendFileID) Export() *LogSendFileIDInternal__ {
	tmp := ((uint64)(l))
	return ((*LogSendFileIDInternal__)(&tmp))
}
func (l LogSendFileIDInternal__) Import() LogSendFileID {
	tmp := (uint64)(l)
	return LogSendFileID((func(x *uint64) (ret uint64) {
		if x == nil {
			return ret
		}
		return *x
	})(&tmp))
}

func (l *LogSendFileID) Encode(enc rpc.Encoder) error {
	return enc.Encode(l.Export())
}

func (l *LogSendFileID) Decode(dec rpc.Decoder) error {
	var tmp LogSendFileIDInternal__
	err := dec.Decode(&tmp)
	if err != nil {
		return err
	}
	*l = tmp.Import()
	return nil
}

func (l LogSendFileID) Bytes() []byte {
	return nil
}

type LogSendBlob []byte
type LogSendBlobInternal__ []byte

func (l LogSendBlob) Export() *LogSendBlobInternal__ {
	tmp := (([]byte)(l))
	return ((*LogSendBlobInternal__)(&tmp))
}
func (l LogSendBlobInternal__) Import() LogSendBlob {
	tmp := ([]byte)(l)
	return LogSendBlob((func(x *[]byte) (ret []byte) {
		if x == nil {
			return ret
		}
		return *x
	})(&tmp))
}

func (l *LogSendBlob) Encode(enc rpc.Encoder) error {
	return enc.Encode(l.Export())
}

func (l *LogSendBlob) Decode(dec rpc.Decoder) error {
	var tmp LogSendBlobInternal__
	err := dec.Decode(&tmp)
	if err != nil {
		return err
	}
	*l = tmp.Import()
	return nil
}

func (l LogSendBlob) Bytes() []byte {
	return (l)[:]
}

var LogSendProtocolID rpc.ProtocolUniqueID = rpc.ProtocolUniqueID(0xfa4726d9)

type LogSendInitArg struct {
}
type LogSendInitArgInternal__ struct {
	_struct struct{} `codec:",toarray"` //lint:ignore U1000 msgpack internal field
}

func (l LogSendInitArgInternal__) Import() LogSendInitArg {
	return LogSendInitArg{}
}
func (l LogSendInitArg) Export() *LogSendInitArgInternal__ {
	return &LogSendInitArgInternal__{}
}
func (l *LogSendInitArg) Encode(enc rpc.Encoder) error {
	return enc.Encode(l.Export())
}

func (l *LogSendInitArg) Decode(dec rpc.Decoder) error {
	var tmp LogSendInitArgInternal__
	err := dec.Decode(&tmp)
	if err != nil {
		return err
	}
	*l = tmp.Import()
	return nil
}

func (l *LogSendInitArg) Bytes() []byte { return nil }

type LogSendInitFileArg struct {
	Id      lib.LogSendID
	FileID  LogSendFileID
	Name    lib.LocalFSPath
	Len     lib.Size
	Hash    lib.StdHash
	NBlocks uint64
}
type LogSendInitFileArgInternal__ struct {
	_struct struct{} `codec:",toarray"` //lint:ignore U1000 msgpack internal field
	Id      *lib.LogSendIDInternal__
	FileID  *LogSendFileIDInternal__
	Name    *lib.LocalFSPathInternal__
	Len     *lib.SizeInternal__
	Hash    *lib.StdHashInternal__
	NBlocks *uint64
}

func (l LogSendInitFileArgInternal__) Import() LogSendInitFileArg {
	return LogSendInitFileArg{
		Id: (func(x *lib.LogSendIDInternal__) (ret lib.LogSendID) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.Id),
		FileID: (func(x *LogSendFileIDInternal__) (ret LogSendFileID) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.FileID),
		Name: (func(x *lib.LocalFSPathInternal__) (ret lib.LocalFSPath) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.Name),
		Len: (func(x *lib.SizeInternal__) (ret lib.Size) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.Len),
		Hash: (func(x *lib.StdHashInternal__) (ret lib.StdHash) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.Hash),
		NBlocks: (func(x *uint64) (ret uint64) {
			if x == nil {
				return ret
			}
			return *x
		})(l.NBlocks),
	}
}
func (l LogSendInitFileArg) Export() *LogSendInitFileArgInternal__ {
	return &LogSendInitFileArgInternal__{
		Id:      l.Id.Export(),
		FileID:  l.FileID.Export(),
		Name:    l.Name.Export(),
		Len:     l.Len.Export(),
		Hash:    l.Hash.Export(),
		NBlocks: &l.NBlocks,
	}
}
func (l *LogSendInitFileArg) Encode(enc rpc.Encoder) error {
	return enc.Encode(l.Export())
}

func (l *LogSendInitFileArg) Decode(dec rpc.Decoder) error {
	var tmp LogSendInitFileArgInternal__
	err := dec.Decode(&tmp)
	if err != nil {
		return err
	}
	*l = tmp.Import()
	return nil
}

func (l *LogSendInitFileArg) Bytes() []byte { return nil }

type LogSendUploadBlockArg struct {
	Id      lib.LogSendID
	FileID  LogSendFileID
	BlockNo uint64
	Block   LogSendBlob
}
type LogSendUploadBlockArgInternal__ struct {
	_struct struct{} `codec:",toarray"` //lint:ignore U1000 msgpack internal field
	Id      *lib.LogSendIDInternal__
	FileID  *LogSendFileIDInternal__
	BlockNo *uint64
	Block   *LogSendBlobInternal__
}

func (l LogSendUploadBlockArgInternal__) Import() LogSendUploadBlockArg {
	return LogSendUploadBlockArg{
		Id: (func(x *lib.LogSendIDInternal__) (ret lib.LogSendID) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.Id),
		FileID: (func(x *LogSendFileIDInternal__) (ret LogSendFileID) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.FileID),
		BlockNo: (func(x *uint64) (ret uint64) {
			if x == nil {
				return ret
			}
			return *x
		})(l.BlockNo),
		Block: (func(x *LogSendBlobInternal__) (ret LogSendBlob) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.Block),
	}
}
func (l LogSendUploadBlockArg) Export() *LogSendUploadBlockArgInternal__ {
	return &LogSendUploadBlockArgInternal__{
		Id:      l.Id.Export(),
		FileID:  l.FileID.Export(),
		BlockNo: &l.BlockNo,
		Block:   l.Block.Export(),
	}
}
func (l *LogSendUploadBlockArg) Encode(enc rpc.Encoder) error {
	return enc.Encode(l.Export())
}

func (l *LogSendUploadBlockArg) Decode(dec rpc.Decoder) error {
	var tmp LogSendUploadBlockArgInternal__
	err := dec.Decode(&tmp)
	if err != nil {
		return err
	}
	*l = tmp.Import()
	return nil
}

func (l *LogSendUploadBlockArg) Bytes() []byte { return nil }

type LogSendInterface interface {
	LogSendInit(context.Context) (lib.LogSendID, error)
	LogSendInitFile(context.Context, LogSendInitFileArg) error
	LogSendUploadBlock(context.Context, LogSendUploadBlockArg) error
	ErrorWrapper() func(error) lib.Status
	CheckArgHeader(ctx context.Context, h lib.Header) error
	MakeResHeader() lib.Header
}

func LogSendMakeGenericErrorWrapper(f LogSendErrorWrapper) rpc.WrapErrorFunc {
	return func(err error) interface{} {
		if err == nil {
			return err
		}
		return f(err).Export()
	}
}

type LogSendErrorUnwrapper func(lib.Status) error
type LogSendErrorWrapper func(error) lib.Status

type logSendErrorUnwrapperAdapter struct {
	h LogSendErrorUnwrapper
}

func (l logSendErrorUnwrapperAdapter) MakeArg() interface{} {
	return &lib.StatusInternal__{}
}

func (l logSendErrorUnwrapperAdapter) UnwrapError(raw interface{}) (appError error, dispatchError error) {
	sTmp, ok := raw.(*lib.StatusInternal__)
	if !ok {
		return nil, errors.New("error converting to internal type in UnwrapError")
	}
	if sTmp == nil {
		return nil, nil
	}
	return l.h(sTmp.Import()), nil
}

var _ rpc.ErrorUnwrapper = logSendErrorUnwrapperAdapter{}

type LogSendClient struct {
	Cli            rpc.GenericClient
	ErrorUnwrapper LogSendErrorUnwrapper
	MakeArgHeader  func() lib.Header
	CheckResHeader func(context.Context, lib.Header) error
}

func (c LogSendClient) LogSendInit(ctx context.Context) (res lib.LogSendID, err error) {
	var arg LogSendInitArg
	warg := &rpc.DataWrap[lib.Header, *LogSendInitArgInternal__]{
		Data: arg.Export(),
	}
	if c.MakeArgHeader != nil {
		warg.Header = c.MakeArgHeader()
	}
	var tmp rpc.DataWrap[lib.Header, lib.LogSendIDInternal__]
	err = c.Cli.Call2(ctx, rpc.NewMethodV2(LogSendProtocolID, 1, "LogSend.logSendInit"), warg, &tmp, 0*time.Millisecond, logSendErrorUnwrapperAdapter{h: c.ErrorUnwrapper})
	if err != nil {
		return
	}
	if c.CheckResHeader != nil {
		err = c.CheckResHeader(ctx, tmp.Header)
		if err != nil {
			return
		}
	}
	res = tmp.Data.Import()
	return
}
func (c LogSendClient) LogSendInitFile(ctx context.Context, arg LogSendInitFileArg) (err error) {
	warg := &rpc.DataWrap[lib.Header, *LogSendInitFileArgInternal__]{
		Data: arg.Export(),
	}
	if c.MakeArgHeader != nil {
		warg.Header = c.MakeArgHeader()
	}
	var tmp rpc.DataWrap[lib.Header, interface{}]
	err = c.Cli.Call2(ctx, rpc.NewMethodV2(LogSendProtocolID, 2, "LogSend.logSendInitFile"), warg, &tmp, 0*time.Millisecond, logSendErrorUnwrapperAdapter{h: c.ErrorUnwrapper})
	if err != nil {
		return
	}
	if c.CheckResHeader != nil {
		err = c.CheckResHeader(ctx, tmp.Header)
		if err != nil {
			return
		}
	}
	return
}
func (c LogSendClient) LogSendUploadBlock(ctx context.Context, arg LogSendUploadBlockArg) (err error) {
	warg := &rpc.DataWrap[lib.Header, *LogSendUploadBlockArgInternal__]{
		Data: arg.Export(),
	}
	if c.MakeArgHeader != nil {
		warg.Header = c.MakeArgHeader()
	}
	var tmp rpc.DataWrap[lib.Header, interface{}]
	err = c.Cli.Call2(ctx, rpc.NewMethodV2(LogSendProtocolID, 3, "LogSend.logSendUploadBlock"), warg, &tmp, 0*time.Millisecond, logSendErrorUnwrapperAdapter{h: c.ErrorUnwrapper})
	if err != nil {
		return
	}
	if c.CheckResHeader != nil {
		err = c.CheckResHeader(ctx, tmp.Header)
		if err != nil {
			return
		}
	}
	return
}
func LogSendProtocol(i LogSendInterface) rpc.ProtocolV2 {
	return rpc.ProtocolV2{
		Name: "LogSend",
		ID:   LogSendProtocolID,
		Methods: map[rpc.Position]rpc.ServeHandlerDescriptionV2{
			1: {
				ServeHandlerDescription: rpc.ServeHandlerDescription{
					MakeArg: func() interface{} {
						var ret rpc.DataWrap[lib.Header, *LogSendInitArgInternal__]
						return &ret
					},
					Handler: func(ctx context.Context, args interface{}) (interface{}, error) {
						typedWrappedArg, ok := args.(*rpc.DataWrap[lib.Header, *LogSendInitArgInternal__])
						if !ok {
							err := rpc.NewTypeError((*rpc.DataWrap[lib.Header, *LogSendInitArgInternal__])(nil), args)
							return nil, err
						}
						if err := i.CheckArgHeader(ctx, typedWrappedArg.Header); err != nil {
							return nil, err
						}
						tmp, err := i.LogSendInit(ctx)
						if err != nil {
							return nil, err
						}
						ret := rpc.DataWrap[lib.Header, *lib.LogSendIDInternal__]{
							Data:   tmp.Export(),
							Header: i.MakeResHeader(),
						}
						return &ret, nil
					},
				},
				Name: "logSendInit",
			},
			2: {
				ServeHandlerDescription: rpc.ServeHandlerDescription{
					MakeArg: func() interface{} {
						var ret rpc.DataWrap[lib.Header, *LogSendInitFileArgInternal__]
						return &ret
					},
					Handler: func(ctx context.Context, args interface{}) (interface{}, error) {
						typedWrappedArg, ok := args.(*rpc.DataWrap[lib.Header, *LogSendInitFileArgInternal__])
						if !ok {
							err := rpc.NewTypeError((*rpc.DataWrap[lib.Header, *LogSendInitFileArgInternal__])(nil), args)
							return nil, err
						}
						if err := i.CheckArgHeader(ctx, typedWrappedArg.Header); err != nil {
							return nil, err
						}
						typedArg := typedWrappedArg.Data
						err := i.LogSendInitFile(ctx, (typedArg.Import()))
						if err != nil {
							return nil, err
						}
						ret := rpc.DataWrap[lib.Header, interface{}]{
							Header: i.MakeResHeader(),
						}
						return &ret, nil
					},
				},
				Name: "logSendInitFile",
			},
			3: {
				ServeHandlerDescription: rpc.ServeHandlerDescription{
					MakeArg: func() interface{} {
						var ret rpc.DataWrap[lib.Header, *LogSendUploadBlockArgInternal__]
						return &ret
					},
					Handler: func(ctx context.Context, args interface{}) (interface{}, error) {
						typedWrappedArg, ok := args.(*rpc.DataWrap[lib.Header, *LogSendUploadBlockArgInternal__])
						if !ok {
							err := rpc.NewTypeError((*rpc.DataWrap[lib.Header, *LogSendUploadBlockArgInternal__])(nil), args)
							return nil, err
						}
						if err := i.CheckArgHeader(ctx, typedWrappedArg.Header); err != nil {
							return nil, err
						}
						typedArg := typedWrappedArg.Data
						err := i.LogSendUploadBlock(ctx, (typedArg.Import()))
						if err != nil {
							return nil, err
						}
						ret := rpc.DataWrap[lib.Header, interface{}]{
							Header: i.MakeResHeader(),
						}
						return &ret, nil
					},
				},
				Name: "logSendUploadBlock",
			},
		},
		WrapError: LogSendMakeGenericErrorWrapper(i.ErrorWrapper()),
	}
}

func init() {
	rpc.AddUnique(LogSendProtocolID)
}
