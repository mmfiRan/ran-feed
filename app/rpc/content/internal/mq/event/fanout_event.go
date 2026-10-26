package event

import "encoding/json"

// FanOutBatch 一次扇出的粉丝分批,由 feed 消费者按粉丝游标切批投递,扇出消费者逐批写收件箱
type FanOutBatch struct {
	ContentID   int64   `json:"content_id"`
	AuthorID    int64   `json:"author_id"`
	PublishedAt int64   `json:"published_at"`
	FollowerIDs []int64 `json:"follower_ids"`
}

func (e *FanOutBatch) Marshal() (string, error) {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func UnmarshalFanOutBatch(data string) (*FanOutBatch, error) {
	var e FanOutBatch
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return nil, err
	}
	return &e, nil
}
