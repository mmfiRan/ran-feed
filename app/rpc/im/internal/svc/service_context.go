package svc

import (
	"ran-feed/app/rpc/im/internal/common/component/impush"
	"ran-feed/app/rpc/im/internal/common/component/imseq"
	"ran-feed/app/rpc/im/internal/config"
	"ran-feed/app/rpc/im/internal/entity/query"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/repositories/conversationrepo"
	"ran-feed/app/rpc/im/internal/repositories/grouprepo"
	"ran-feed/app/rpc/im/internal/repositories/memberrepo"
	"ran-feed/app/rpc/im/internal/repositories/messagerepo"
	"ran-feed/pkg/orm"
)

// ServiceContext 依赖容器
type ServiceContext struct {
	Config                 config.Config
	ConversationRepository repositories.ConversationRepository
	MemberRepository       repositories.MemberRepository
	MessageRepository      repositories.MessageRepository
	GroupRepository        repositories.GroupRepository
	Seq                    *imseq.Allocator
	Push                   *impush.Notifier
}

// NewServiceContext 组装依赖
func NewServiceContext(c config.Config) *ServiceContext {
	// query.SetDefault 必须在使用任何 Repository 之前只执行一次
	mysql := orm.MustNewMysql(&orm.Config{DSN: c.MySQL.DataSource})
	query.SetDefault(mysql.DB)

	return &ServiceContext{
		Config:                 c,
		ConversationRepository: conversationrepo.New(),
		MemberRepository:       memberrepo.New(),
		MessageRepository:      messagerepo.New(),
		GroupRepository:        grouprepo.New(),
		Seq:                    imseq.New(),
		Push:                   impush.New(c.RedisConfig),
	}
}
