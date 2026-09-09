package admincontentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/utils/contentcache"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminSetContentStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
}

func NewAdminSetContentStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSetContentStatusLogic {
	return &AdminSetContentStatusLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
}

// AdminSetContentStatus 下架/恢复 本轮仅支持两态互转
//   - 下架 PUBLISHED -> TAKEN_DOWN
//   - 恢复 TAKEN_DOWN -> PUBLISHED
//
// 状态翻转后失效二级缓存 feed 读路径 miss 回源经 BatchGetPublishedByIDs 过滤 下架内容自动从各流消失
func (l *AdminSetContentStatusLogic) AdminSetContentStatus(in *content.AdminSetContentStatusReq) (*content.AdminSetContentStatusRes, error) {
	if in == nil || in.ContentId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	target := in.GetStatus()
	if target != content.ContentStatus_TAKEN_DOWN && target != content.ContentStatus_PUBLISHED {
		return nil, errorx.NewMsg("不支持的目标状态")
	}

	row, err := l.contentRepo.AdminGetByID(in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("内容不存在")
	}

	cur := content.ContentStatus(row.Status)
	noop, err := validateStatusTransition(cur, target)
	if err != nil {
		return nil, err
	}
	if noop {
		return &content.AdminSetContentStatusRes{Status: target}, nil
	}

	if _, err = l.contentRepo.AdminUpdateStatus(in.ContentId, int32(target), in.GetOperatorId()); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新内容状态失败"))
	}

	// 失效二级缓存 失败不阻断 靠 TTL 收敛
	if err = contentcache.Invalidate(l.ctx, l.svcCtx.Redis, in.ContentId); err != nil {
		l.Errorf("失效内容详情二级缓存失败 contentID=%d err=%v", in.ContentId, err)
	}

	return &content.AdminSetContentStatusRes{Status: target}, nil
}

// validateStatusTransition 校验下架/恢复状态机 返回 noop 表示当前已是目标态无需落库
//   - 下架 TAKEN_DOWN 仅允许从 PUBLISHED 转入
//   - 恢复 PUBLISHED 仅允许从 TAKEN_DOWN 转入
//
// 杜绝越权把草稿/待审/拒绝直接改成发布态
func validateStatusTransition(cur, target content.ContentStatus) (noop bool, err error) {
	if cur == target {
		return true, nil
	}
	switch target {
	case content.ContentStatus_TAKEN_DOWN:
		if cur != content.ContentStatus_PUBLISHED {
			return false, errorx.NewMsg("仅已发布内容可下架")
		}
	case content.ContentStatus_PUBLISHED:
		if cur != content.ContentStatus_TAKEN_DOWN {
			return false, errorx.NewMsg("仅已下架内容可恢复")
		}
	default:
		return false, errorx.NewMsg("不支持的目标状态")
	}
	return false, nil
}
