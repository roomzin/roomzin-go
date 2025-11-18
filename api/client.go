package api

import "github.com/roomzin/roomzin-go/types"

type CacheClientAPI interface {
	GetCodecs() (*types.Codecs, error)
	SetProp(p types.SetPropPayload) error
	SearchProp(p types.SearchPropPayload) ([]string, error)
	SearchAvail(p types.SearchAvailPayload) ([]types.PropertyAvail, error)
	SetRoomPkg(p types.SetRoomPkgPayload) error
	SetRoomAvl(p types.UpdRoomAvlPayload) (uint8, error)
	IncRoomAvl(p types.UpdRoomAvlPayload) (uint8, error)
	DecRoomAvl(p types.UpdRoomAvlPayload) (uint8, error)
	PropExist(propertyID string) (bool, error)
	PropRoomExist(p types.PropRoomExistPayload) (bool, error)
	PropRoomList(propertyID string) ([]string, error)
	PropRoomDateList(p types.PropRoomDateListPayload) ([]string, error)
	DelProp(propertyID string) error
	DelSegment(segment string) error
	DelPropDay(p types.DelPropDayRequest) error
	DelPropRoom(p types.DelPropRoomPayload) error
	DelRoomDay(p types.DelRoomDayRequest) error
	GetPropRoomDay(p types.GetRoomDayRequest) (types.GetRoomDayResult, error)
	GetSegments() ([]types.SegmentInfo, error)
	Close() error
}
