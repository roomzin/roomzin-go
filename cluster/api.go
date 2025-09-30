package cluster

import (
	"context"
	"errors"
	"time"

	"github.com/roomzin/roomzin-go/api"
	"github.com/roomzin/roomzin-go/internal/cluster"
	"github.com/roomzin/roomzin-go/internal/command"
	"github.com/roomzin/roomzin-go/types"
)

type client struct {
	handler *cluster.Handler
	cfg     *ClusterConfig
	ctx     context.Context
	cancel  context.CancelFunc
}

func New(cfg *ClusterConfig) (api.CacheClientAPI, error) {
	if cfg == nil {
		return nil, errors.New("cfg must not be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	icfg := &cluster.Config{
		SeedHosts:         cfg.SeedHosts,
		APIPort:           cfg.APIPort,
		TCPPort:           cfg.TCPPort,
		AuthToken:         cfg.AuthToken,
		Timeout:           cfg.Timeout,
		HttpTimeout:       cfg.HttpTimeout,
		KeepAlive:         cfg.KeepAlive,
		MaxActiveConns:    cfg.MaxActiveConns,
		NodeProbeInterval: 2 * time.Second,
	}

	clusterClient := cluster.NewHandler(icfg)
	clusterClient.Start(ctx)

	return &client{
		handler: clusterClient,
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

func (c *client) Close() error {
	c.cancel()
	return nil
}

// --------------------------------------------------
//
//	public API
//
// --------------------------------------------------

/* ----------  READ helpers (follower)  ---------- */
func (c *client) SearchProp(p types.SearchPropPayload) ([]string, error) {
	req, err := command.BuildSearchPropPayload(p)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, false, req)
	if err != nil {
		return nil, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return nil, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseSearchPropResp(resp.Status, resp.Fields)
}

func (c *client) SearchAvail(p types.SearchAvailPayload) ([]types.PropertyAvail, error) {
	req, err := command.BuildSearchAvailPayload(p)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, false, req)
	if err != nil {
		return nil, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return nil, errors.New(string(resp.Fields[1].Data))
	}

	result, err := command.ParseSearchAvailResp(resp.Status, resp.Fields)
	return result, err
}

func (c *client) PropExist(propertyID string) (bool, error) {
	req, err := command.BuildPropExistPayload(propertyID)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, false, req)
	if err != nil {
		return false, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return false, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParsePropExistResp(resp.Status, resp.Fields)
}

func (c *client) PropRoomExist(p types.PropRoomExistPayload) (bool, error) {
	req, err := command.BuildPropRoomExistPayload(p)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, false, req)
	if err != nil {
		return false, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return false, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParsePropRoomExistResp(resp.Status, resp.Fields)
}

func (c *client) PropRoomList(propertyID string) ([]string, error) {
	req, err := command.BuildPropRoomListPayload(propertyID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, false, req)
	if err != nil {
		return nil, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return nil, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParsePropRoomListResp(resp.Status, resp.Fields)
}

func (c *client) PropRoomDateList(p types.PropRoomDateListPayload) ([]string, error) {
	req, err := command.BuildPropRoomDateListPayload(p)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, false, req)
	if err != nil {
		return nil, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return nil, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParsePropRoomDateListResp(resp.Status, resp.Fields)
}

func (c *client) GetPropRoomDay(p types.GetRoomDayRequest) (types.GetRoomDayResult, error) {
	req, err := command.BuildGetPropRoomDayPayload(p)
	if err != nil {
		return types.GetRoomDayResult{}, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, false, req)
	if err != nil {
		return types.GetRoomDayResult{}, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return types.GetRoomDayResult{}, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseGetPropRoomDayResp(resp.Status, resp.Fields)
}

/* ----------  WRITE helpers (leader)  ---------- */

func (c *client) SetProp(p types.SetPropPayload) error {
	req, err := command.BuildSetPropPayload(p)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseSetPropResp(resp.Status, resp.Fields)
}

func (c *client) SetRoomPkg(p types.SetRoomPkgPayload) error {
	req, err := command.BuildSetRoomPkgPayload(p)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseSetRoomPkgResp(resp.Status, resp.Fields)
}

func (c *client) SetRoomAvl(p types.UpdRoomAvlPayload) (uint8, error) {
	req, err := command.BuildSetRoomAvlPayload(p)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return 0, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return 0, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseSetRoomAvlResp(resp.Status, resp.Fields)
}

func (c *client) IncRoomAvl(p types.UpdRoomAvlPayload) (uint8, error) {
	req, err := command.BuildIncRoomAvlPayload(p)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return 0, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return 0, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseIncRoomAvlResp(resp.Status, resp.Fields)
}

func (c *client) DecRoomAvl(p types.UpdRoomAvlPayload) (uint8, error) {
	req, err := command.BuildDecRoomAvlPayload(p)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return 0, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return 0, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseDecRoomAvlResp(resp.Status, resp.Fields)
}

func (c *client) DelProp(propertyID string) error {
	req, err := command.BuildDelPropPayload(propertyID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseDelPropResp(resp.Status, resp.Fields)
}

func (c *client) DelSegment(segment string) error {
	req, err := command.BuildDelSegmentPayload(segment)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseDelSegmentResp(resp.Status, resp.Fields)
}

func (c *client) DelPropDay(p types.DelPropDayRequest) error {
	req, err := command.BuildDelPropDayPayload(p)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseDelPropDayResp(resp.Status, resp.Fields)
}

func (c *client) DelPropRoom(p types.DelPropRoomPayload) error {
	req, err := command.BuildDelPropRoomPayload(p)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseDelPropRoomResp(resp.Status, resp.Fields)
}

func (c *client) DelRoomDay(p types.DelRoomDayRequest) error {
	req, err := command.BuildDelRoomDayPayload(p)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseDelRoomDayResp(resp.Status, resp.Fields)
}

/* ----------  MISC  ---------- */

func (c *client) SaveSnapshot() error {
	// any node can trigger snapshot
	req, err := command.BuildSaveSnapshotPayload()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, false, req)
	if err != nil {
		return err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseSaveSnapshotResp(resp.Status, resp.Fields)
}

func (c *client) GetSegments() ([]types.SegmentInfo, error) {
	req, err := command.BuildGetSegmentsPayload()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.handler.Execute(ctx, true, req)
	if err != nil {
		return nil, err
	}
	if resp.Status == "ERROR" && len(resp.Fields) > 0 {
		return nil, errors.New(string(resp.Fields[1].Data))
	}

	return command.ParseGetSegmentsResp(resp.Status, resp.Fields)
}
