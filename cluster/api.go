package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	codecs  *types.Codecs
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

	c := &client{
		handler: clusterClient,
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
	}

	c.handler.SetOnReconnectCallback(func() {
		c.codecs = nil
	})

	var err error
	c.codecs, err = c.fetchCodecs()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *client) getCodecs() *types.Codecs {
	if c.codecs != nil {
		return c.codecs
	}
	c.codecs, _ = c.fetchCodecs()
	return c.codecs
}

func (c *client) fetchCodecs() (*types.Codecs, error) {
	req, err := command.BuildGetCodecsPayload()
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

	return command.ParseGetCodecsResp(resp.Status, resp.Fields)
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

func (c *client) GetCodecs() (*types.Codecs, error) {
	if c.codecs != nil {
		return c.codecs, nil
	}
	var err error
	c.codecs, err = c.fetchCodecs()
	if err != nil {
		return nil, err
	}
	return c.codecs, nil
}

/* ----------  READ helpers (follower)  ---------- */
func (c *client) SearchProp(p types.SearchPropPayload) ([]string, error) {
	if ok, errMsg := p.Verify(c.getCodecs()); !ok {
		return nil, fmt.Errorf("invalid SearchProp payload: %s", errMsg)
	}
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
	if ok, errMsg := p.Verify(c.getCodecs()); !ok {
		return nil, fmt.Errorf("invalid SearchAvail payload: %s", errMsg)
	}
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

	result, err := command.ParseSearchAvailResp(c.getCodecs(), resp.Status, resp.Fields)
	return result, err
}

func (c *client) PropExist(propertyID string) (bool, error) {
	if strings.TrimSpace(propertyID) == "" {
		return false, fmt.Errorf("propertyID is required")
	}
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
	if ok, errMsg := p.Verify(); !ok {
		return false, fmt.Errorf("invalid PropRoomExist payload: %s", errMsg)
	}
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
	if strings.TrimSpace(propertyID) == "" {
		return nil, fmt.Errorf("propertyID is required")
	}
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
	if ok, errMsg := p.Verify(); !ok {
		return nil, fmt.Errorf("invalid PropRoomDateList payload: %s", errMsg)
	}
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
	if ok, errMsg := p.Verify(); !ok {
		return types.GetRoomDayResult{}, fmt.Errorf("invalid GetPropRoomDay payload: %s", errMsg)
	}
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

	return command.ParseGetPropRoomDayResp(c.getCodecs(), resp.Status, resp.Fields)
}

/* ----------  WRITE helpers (leader)  ---------- */

func (c *client) SetProp(p types.SetPropPayload) error {
	if ok, errMsg := p.Verify(c.getCodecs()); !ok {
		return fmt.Errorf("invalid SetProp payload: %s", errMsg)
	}
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
	if ok, errMsg := p.Verify(c.getCodecs()); !ok {
		return fmt.Errorf("invalid SetRoomPkg payload: %s", errMsg)
	}
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
	if ok, errMsg := p.Verify(); !ok {
		return 0, fmt.Errorf("invalid SetRoomAvl payload: %s", errMsg)
	}
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
	if ok, errMsg := p.Verify(); !ok {
		return 0, fmt.Errorf("invalid IncRoomAvl payload: %s", errMsg)
	}
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
	if ok, errMsg := p.Verify(); !ok {
		return 0, fmt.Errorf("invalid DecRoomAvl payload: %s", errMsg)
	}
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
	if strings.TrimSpace(propertyID) == "" {
		return fmt.Errorf("propertyID is required")
	}
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
	if strings.TrimSpace(segment) == "" {
		return fmt.Errorf("segment is required")
	}
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
	if ok, errMsg := p.Verify(); !ok {
		return fmt.Errorf("invalid DelPropDay payload: %s", errMsg)
	}
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
	if ok, errMsg := p.Verify(); !ok {
		return fmt.Errorf("invalid DelPropRoom payload: %s", errMsg)
	}
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
	if ok, errMsg := p.Verify(); !ok {
		return fmt.Errorf("invalid DelRoomDay payload: %s", errMsg)
	}
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
func (c *client) GetSegments() ([]types.SegmentInfo, error) {
	req, err := command.BuildGetSegmentsPayload()
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

	return command.ParseGetSegmentsResp(resp.Status, resp.Fields)
}
