package rtsp

import (
	"SM/Model"
	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/base"
	"github.com/gogf/gf/v2/text/gstr"
	. "m7s.live/engine/v4"
)

type RTSPIO struct {
	tracks       gortsplib.Tracks
	stream       *gortsplib.ServerStream
	audioTrackId int
	videoTrackId int
}

func (conf *RTSPConfig) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	RTSPPlugin.Debug("conn opened")
}

func (conf *RTSPConfig) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	RTSPPlugin.Debug("conn closed")
	if p, ok := conf.LoadAndDelete(ctx.Conn); ok {
		p.(IIO).Stop()
	}
}

func (conf *RTSPConfig) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	RTSPPlugin.Debug("session opened")
}

func (conf *RTSPConfig) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	RTSPPlugin.Debug("session closed")
	if p, ok := conf.LoadAndDelete(ctx.Session); ok {
		p.(IIO).Stop()
	}
}

// called after receiving a DESCRIBE request.
func (conf *RTSPConfig) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	RTSPPlugin.Debug("describe request")
	var suber RTSPSubscriber
	suber.SetIO(ctx.Conn.NetConn())
	paths := gstr.Split(ctx.Path, "/")
	if len(paths) > 1 && paths[0] == "fjrh" {
		if Model.Lives.Contains(paths[1]) {
			liveInfo := Model.Lives.Get(paths[1]).(*Model.LiveInfo)
			ctx.Path = liveInfo.App + "/" + liveInfo.StreamID
			liveInfo.Subscribers = append(liveInfo.Subscribers, &suber)
		}
	}
	if err := RTSPPlugin.Subscribe(ctx.Path, &suber); err == nil {
		conf.Store(ctx.Conn, &suber)
		return &base.Response{
			StatusCode: base.StatusOK,
		}, suber.stream, nil
	} else {
		return nil, nil, err
	}
}

func (conf *RTSPConfig) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	var resp base.Response
	resp.StatusCode = base.StatusOK
	if p, ok := conf.Load(ctx.Conn); ok {
		switch v := p.(type) {
		case *RTSPSubscriber:
			return &resp, v.stream, nil
		case *RTSPPublisher:
			return &resp, v.stream, nil
		}
	}
	resp.StatusCode = base.StatusNotFound
	return &resp, nil, nil
}

func (conf *RTSPConfig) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	var resp base.Response
	resp.StatusCode = base.StatusNotFound
	if p, ok := conf.Load(ctx.Conn); ok {
		switch v := p.(type) {
		case *RTSPSubscriber:
			resp.StatusCode = base.StatusOK
			go func() {
				v.PlayRTP()
				ctx.Session.Close()
			}()
		}
	}
	return &resp, nil
}
func (conf *RTSPConfig) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}
func (conf *RTSPConfig) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	p := &RTSPPublisher{}
	p.SetIO(ctx.Conn.NetConn())
	if err := RTSPPlugin.Publish(ctx.Path, p); err == nil {
		p.tracks = ctx.Tracks
		p.stream = gortsplib.NewServerStream(ctx.Tracks)
		if err = p.SetTracks(); err != nil {
			return nil, err
		}
		conf.Store(ctx.Conn, p)
		conf.Store(ctx.Session, p)
	} else {
		return &base.Response{
			StatusCode: base.StatusBadRequest,
		}, err
	}
	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (conf *RTSPConfig) OnPacketRTP(ctx *gortsplib.ServerHandlerOnPacketRTPCtx) {
	if p, ok := conf.Load(ctx.Session); ok {
		switch v := p.(type) {
		case *RTSPPublisher:
			v.Tracks[ctx.TrackID].WriteRTPPack(ctx.Packet)
		}
	}
}
