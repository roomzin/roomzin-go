package single

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/roomzin/roomzin-go/api"
	"github.com/roomzin/roomzin-go/internal/command"
	"github.com/roomzin/roomzin-go/internal/single"
	"github.com/roomzin/roomzin-go/types"
)

type client struct {
	handler *single.Handler
	cfg     *Config
	ctx     context.Context
	cancel  context.CancelFunc
	codecs  *types.Codecs
}

func New(cfg *Config) (api.CacheClientAPI, error) {
	if cfg == nil {
		return nil, errors.New("cfg must not be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	icfg := &single.Config{
		Addr:      cfg.Host,
		TCPPort:   cfg.TCPPort,
		AuthToken: cfg.AuthToken,
		Timeout:   cfg.Timeout,
		KeepAlive: cfg.KeepAlive,
	}

	singleClient, err := single.NewHandler(icfg, ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	c := &client{
		handler: singleClient,
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
	}

	c.handler.OnReconnect = func() {
		c.codecs = nil
	}

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
	payload, _ := command.BuildGetCodecsPayload()
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return nil, errors.New("failed to get codecs list from server")
	}
	return command.ParseGetCodecsResp(res.Status, res.Fields)
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

func (c *client) SetProp(p types.SetPropPayload) error {
	if err := p.Verify(c.getCodecs()); err != nil {
		return err
	}
	payload, _ := command.BuildSetPropPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return err
	}
	return command.ParseSetPropResp(res.Status, res.Fields)
}

func (c *client) SearchProp(p types.SearchPropPayload) ([]string, error) {
	if err := p.Verify(c.getCodecs()); err != nil {
		return nil, err
	}
	payload, _ := command.BuildSearchPropPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return nil, err
	}
	return command.ParseSearchPropResp(res.Status, res.Fields)
}

func (c *client) SearchAvail(p types.SearchAvailPayload) ([]types.PropertyAvail, error) {
	if err := p.Verify(c.getCodecs()); err != nil {
		return nil, err
	}
	payload, _ := command.BuildSearchAvailPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return nil, err
	}
	return command.ParseSearchAvailResp(c.getCodecs(), res.Status, res.Fields)
}

func (c *client) SetRoomPkg(p types.SetRoomPkgPayload) error {
	if err := p.Verify(c.getCodecs()); err != nil {
		return err
	}
	payload, _ := command.BuildSetRoomPkgPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return err
	}
	return command.ParseSetRoomPkgResp(res.Status, res.Fields)
}

func (c *client) SetRoomAvl(p types.UpdRoomAvlPayload) (uint8, error) {
	if err := p.Verify(); err != nil {
		return 0, err
	}
	payload, _ := command.BuildSetRoomAvlPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return 0, err
	}
	return command.ParseSetRoomAvlResp(res.Status, res.Fields)
}

func (c *client) IncRoomAvl(p types.UpdRoomAvlPayload) (uint8, error) {
	if err := p.Verify(); err != nil {
		return 0, err
	}
	payload, _ := command.BuildIncRoomAvlPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return 0, err
	}
	return command.ParseIncRoomAvlResp(res.Status, res.Fields)
}

func (c *client) DecRoomAvl(p types.UpdRoomAvlPayload) (uint8, error) {
	if err := p.Verify(); err != nil {
		return 0, err
	}
	payload, _ := command.BuildDecRoomAvlPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return 0, err
	}
	return command.ParseDecRoomAvlResp(res.Status, res.Fields)
}

func (c *client) PropExist(propertyID string) (bool, error) {
	if strings.TrimSpace(propertyID) == "" {
		return false, fmt.Errorf("VALIDATION_ERROR: propertyID is required")
	}
	payload, _ := command.BuildPropExistPayload(propertyID)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return false, err
	}
	return command.ParsePropExistResp(res.Status, res.Fields)
}

func (c *client) PropRoomExist(p types.PropRoomExistPayload) (bool, error) {
	if err := p.Verify(); err != nil {
		return false, err
	}
	payload, _ := command.BuildPropRoomExistPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return false, err
	}
	return command.ParsePropRoomExistResp(res.Status, res.Fields)
}

func (c *client) PropRoomList(propertyID string) ([]string, error) {
	if strings.TrimSpace(propertyID) == "" {
		return nil, fmt.Errorf("VALIDATION_ERROR: propertyID is required")
	}
	payload, _ := command.BuildPropRoomListPayload(propertyID)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return nil, err
	}
	return command.ParsePropRoomListResp(res.Status, res.Fields)
}

func (c *client) PropRoomDateList(p types.PropRoomDateListPayload) ([]string, error) {
	if err := p.Verify(); err != nil {
		return nil, err
	}
	payload, _ := command.BuildPropRoomDateListPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return nil, err
	}
	return command.ParsePropRoomDateListResp(res.Status, res.Fields)
}

func (c *client) DelProp(propertyID string) error {
	if strings.TrimSpace(propertyID) == "" {
		return fmt.Errorf("VALIDATION_ERROR: propertyID is required")
	}
	payload, _ := command.BuildDelPropPayload(propertyID)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return err
	}
	return command.ParseDelPropResp(res.Status, res.Fields)
}

func (c *client) DelSegment(segment string) error {
	if strings.TrimSpace(segment) == "" {
		return fmt.Errorf("VALIDATION_ERROR: segment is required")
	}
	payload, _ := command.BuildDelSegmentPayload(segment)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return err
	}
	return command.ParseDelSegmentResp(res.Status, res.Fields)
}

func (c *client) DelPropDay(p types.DelPropDayRequest) error {
	if err := p.Verify(); err != nil {
		return err
	}
	payload, _ := command.BuildDelPropDayPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return err
	}
	return command.ParseDelPropDayResp(res.Status, res.Fields)
}

func (c *client) DelPropRoom(p types.DelPropRoomPayload) error {
	if err := p.Verify(); err != nil {
		return err
	}
	payload, _ := command.BuildDelPropRoomPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return err
	}
	return command.ParseDelPropRoomResp(res.Status, res.Fields)
}

func (c *client) DelRoomDay(p types.DelRoomDayRequest) error {
	if err := p.Verify(); err != nil {
		return err
	}
	payload, _ := command.BuildDelRoomDayPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return err
	}
	return command.ParseDelRoomDayResp(res.Status, res.Fields)
}

func (c *client) GetPropRoomDay(p types.GetRoomDayRequest) (types.GetRoomDayResult, error) {
	if err := p.Verify(); err != nil {
		return types.GetRoomDayResult{}, err
	}
	payload, _ := command.BuildGetPropRoomDayPayload(p)
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return types.GetRoomDayResult{}, err
	}
	return command.ParseGetPropRoomDayResp(c.getCodecs(), res.Status, res.Fields)
}

func (c *client) GetSegments() ([]types.SegmentInfo, error) {
	payload, _ := command.BuildGetSegmentsPayload()
	res, err := c.handler.RoundTrip(c.handler.NextID(), payload)
	if err != nil {
		return nil, err
	}
	return command.ParseGetSegmentsResp(res.Status, res.Fields)
}
