// Auto-generated to Go types and interfaces using snowpc 0.0.4 (https://github.com/foks-proj/go-snowpack-compiler)
//  Input file:../../proto-src/lcl/logsend.snowp

package lcl

import (
	"context"
	"errors"
	"github.com/foks-proj/go-snowpack-rpc/rpc"
	"time"
)

import lib "github.com/foks-proj/go-foks/proto/lib"

type LogSendSet struct {
	Files []lib.LocalFSPath
}
type LogSendSetInternal__ struct {
	_struct struct{} `codec:",toarray"` //lint:ignore U1000 msgpack internal field
	Files   *[](*lib.LocalFSPathInternal__)
}

func (l LogSendSetInternal__) Import() LogSendSet {
	return LogSendSet{
		Files: (func(x *[](*lib.LocalFSPathInternal__)) (ret []lib.LocalFSPath) {
			if x == nil || len(*x) == 0 {
				return nil
			}
			ret = make([]lib.LocalFSPath, len(*x))
			for k, v := range *x {
				if v == nil {
					continue
				}
				ret[k] = (func(x *lib.LocalFSPathInternal__) (ret lib.LocalFSPath) {
					if x == nil {
						return ret
					}
					return x.Import()
				})(v)
			}
			return ret
		})(l.Files),
	}
}
func (l LogSendSet) Export() *LogSendSetInternal__ {
	return &LogSendSetInternal__{
		Files: (func(x []lib.LocalFSPath) *[](*lib.LocalFSPathInternal__) {
			if len(x) == 0 {
				return nil
			}
			ret := make([](*lib.LocalFSPathInternal__), len(x))
			for k, v := range x {
				ret[k] = v.Export()
			}
			return &ret
		})(l.Files),
	}
}
func (l *LogSendSet) Encode(enc rpc.Encoder) error {
	return enc.Encode(l.Export())
}

func (l *LogSendSet) Decode(dec rpc.Decoder) error {
	var tmp LogSendSetInternal__
	err := dec.Decode(&tmp)
	if err != nil {
		return err
	}
	*l = tmp.Import()
	return nil
}

func (l *LogSendSet) Bytes() []byte { return nil }

type LogSendRes struct {
	Id   lib.LogSendID
	Host lib.TCPAddr
}
type LogSendResInternal__ struct {
	_struct struct{} `codec:",toarray"` //lint:ignore U1000 msgpack internal field
	Id      *lib.LogSendIDInternal__
	Host    *lib.TCPAddrInternal__
}

func (l LogSendResInternal__) Import() LogSendRes {
	return LogSendRes{
		Id: (func(x *lib.LogSendIDInternal__) (ret lib.LogSendID) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.Id),
		Host: (func(x *lib.TCPAddrInternal__) (ret lib.TCPAddr) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.Host),
	}
}
func (l LogSendRes) Export() *LogSendResInternal__ {
	return &LogSendResInternal__{
		Id:   l.Id.Export(),
		Host: l.Host.Export(),
	}
}
func (l *LogSendRes) Encode(enc rpc.Encoder) error {
	return enc.Encode(l.Export())
}

func (l *LogSendRes) Decode(dec rpc.Decoder) error {
	var tmp LogSendResInternal__
	err := dec.Decode(&tmp)
	if err != nil {
		return err
	}
	*l = tmp.Import()
	return nil
}

func (l *LogSendRes) Bytes() []byte { return nil }

var LogSendProtocolID rpc.ProtocolUniqueID = rpc.ProtocolUniqueID(0xfeb332f6)

type LogSendListArg struct {
	N uint64
}
type LogSendListArgInternal__ struct {
	_struct struct{} `codec:",toarray"` //lint:ignore U1000 msgpack internal field
	N       *uint64
}

func (l LogSendListArgInternal__) Import() LogSendListArg {
	return LogSendListArg{
		N: (func(x *uint64) (ret uint64) {
			if x == nil {
				return ret
			}
			return *x
		})(l.N),
	}
}
func (l LogSendListArg) Export() *LogSendListArgInternal__ {
	return &LogSendListArgInternal__{
		N: &l.N,
	}
}
func (l *LogSendListArg) Encode(enc rpc.Encoder) error {
	return enc.Encode(l.Export())
}

func (l *LogSendListArg) Decode(dec rpc.Decoder) error {
	var tmp LogSendListArgInternal__
	err := dec.Decode(&tmp)
	if err != nil {
		return err
	}
	*l = tmp.Import()
	return nil
}

func (l *LogSendListArg) Bytes() []byte { return nil }

type LogSendArg struct {
	Set LogSendSet
}
type LogSendArgInternal__ struct {
	_struct struct{} `codec:",toarray"` //lint:ignore U1000 msgpack internal field
	Set     *LogSendSetInternal__
}

func (l LogSendArgInternal__) Import() LogSendArg {
	return LogSendArg{
		Set: (func(x *LogSendSetInternal__) (ret LogSendSet) {
			if x == nil {
				return ret
			}
			return x.Import()
		})(l.Set),
	}
}
func (l LogSendArg) Export() *LogSendArgInternal__ {
	return &LogSendArgInternal__{
		Set: l.Set.Export(),
	}
}
func (l *LogSendArg) Encode(enc rpc.Encoder) error {
	return enc.Encode(l.Export())
}

func (l *LogSendArg) Decode(dec rpc.Decoder) error {
	var tmp LogSendArgInternal__
	err := dec.Decode(&tmp)
	if err != nil {
		return err
	}
	*l = tmp.Import()
	return nil
}

func (l *LogSendArg) Bytes() []byte { return nil }

type LogSendInterface interface {
	LogSendList(context.Context, uint64) (LogSendSet, error)
	LogSend(context.Context, LogSendSet) (LogSendRes, error)
	ErrorWrapper() func(error) lib.Status
	CheckArgHeader(ctx context.Context, h Header) error
	MakeResHeader() Header
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
	MakeArgHeader  func() Header
	CheckResHeader func(context.Context, Header) error
}

func (c LogSendClient) LogSendList(ctx context.Context, n uint64) (res LogSendSet, err error) {
	arg := LogSendListArg{
		N: n,
	}
	warg := &rpc.DataWrap[Header, *LogSendListArgInternal__]{
		Data: arg.Export(),
	}
	if c.MakeArgHeader != nil {
		warg.Header = c.MakeArgHeader()
	}
	var tmp rpc.DataWrap[Header, LogSendSetInternal__]
	err = c.Cli.Call2(ctx, rpc.NewMethodV2(LogSendProtocolID, 1, "LogSend.logSendList"), warg, &tmp, 0*time.Millisecond, logSendErrorUnwrapperAdapter{h: c.ErrorUnwrapper})
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
func (c LogSendClient) LogSend(ctx context.Context, set LogSendSet) (res LogSendRes, err error) {
	arg := LogSendArg{
		Set: set,
	}
	warg := &rpc.DataWrap[Header, *LogSendArgInternal__]{
		Data: arg.Export(),
	}
	if c.MakeArgHeader != nil {
		warg.Header = c.MakeArgHeader()
	}
	var tmp rpc.DataWrap[Header, LogSendResInternal__]
	err = c.Cli.Call2(ctx, rpc.NewMethodV2(LogSendProtocolID, 2, "LogSend.logSend"), warg, &tmp, 0*time.Millisecond, logSendErrorUnwrapperAdapter{h: c.ErrorUnwrapper})
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
func LogSendProtocol(i LogSendInterface) rpc.ProtocolV2 {
	return rpc.ProtocolV2{
		Name: "LogSend",
		ID:   LogSendProtocolID,
		Methods: map[rpc.Position]rpc.ServeHandlerDescriptionV2{
			1: {
				ServeHandlerDescription: rpc.ServeHandlerDescription{
					MakeArg: func() interface{} {
						var ret rpc.DataWrap[Header, *LogSendListArgInternal__]
						return &ret
					},
					Handler: func(ctx context.Context, args interface{}) (interface{}, error) {
						typedWrappedArg, ok := args.(*rpc.DataWrap[Header, *LogSendListArgInternal__])
						if !ok {
							err := rpc.NewTypeError((*rpc.DataWrap[Header, *LogSendListArgInternal__])(nil), args)
							return nil, err
						}
						if err := i.CheckArgHeader(ctx, typedWrappedArg.Header); err != nil {
							return nil, err
						}
						typedArg := typedWrappedArg.Data
						tmp, err := i.LogSendList(ctx, (typedArg.Import()).N)
						if err != nil {
							return nil, err
						}
						ret := rpc.DataWrap[Header, *LogSendSetInternal__]{
							Data:   tmp.Export(),
							Header: i.MakeResHeader(),
						}
						return &ret, nil
					},
				},
				Name: "logSendList",
			},
			2: {
				ServeHandlerDescription: rpc.ServeHandlerDescription{
					MakeArg: func() interface{} {
						var ret rpc.DataWrap[Header, *LogSendArgInternal__]
						return &ret
					},
					Handler: func(ctx context.Context, args interface{}) (interface{}, error) {
						typedWrappedArg, ok := args.(*rpc.DataWrap[Header, *LogSendArgInternal__])
						if !ok {
							err := rpc.NewTypeError((*rpc.DataWrap[Header, *LogSendArgInternal__])(nil), args)
							return nil, err
						}
						if err := i.CheckArgHeader(ctx, typedWrappedArg.Header); err != nil {
							return nil, err
						}
						typedArg := typedWrappedArg.Data
						tmp, err := i.LogSend(ctx, (typedArg.Import()).Set)
						if err != nil {
							return nil, err
						}
						ret := rpc.DataWrap[Header, *LogSendResInternal__]{
							Data:   tmp.Export(),
							Header: i.MakeResHeader(),
						}
						return &ret, nil
					},
				},
				Name: "logSend",
			},
		},
		WrapError: LogSendMakeGenericErrorWrapper(i.ErrorWrapper()),
	}
}

func init() {
	rpc.AddUnique(LogSendProtocolID)
}
