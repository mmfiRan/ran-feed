// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/transform"
	"ran-feed/pkg/utils"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContentUploadsCredentialsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContentUploadsCredentialsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentUploadsCredentialsLogic {
	return &ContentUploadsCredentialsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContentUploadsCredentialsLogic) ContentUploadsCredentials(req *types.ContentUploadsCredentialsReq) (resp *types.ContentUploadsCredentialsRes, err error) {
	id, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.NewMsg("获取用户ID失败")
	}

	scene, err := transform.ParseEnum[content.ContentUploadsCredentialsReq_Scene](content.ContentUploadsCredentialsReq_Scene_value, *req.Scene)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("上传场景参数错误"))
	}

	fileExt, err := transform.ParseEnum[content.ContentUploadsCredentialsReq_FileExt](content.ContentUploadsCredentialsReq_FileExt_value, *req.FileExt)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("文件扩展名参数错误"))
	}

	rpcReq := &content.ContentUploadsCredentialsReq{
		UserId:   id,
		Scene:    scene,
		FileExt:  fileExt,
		FileSize: *req.FileSize,
		FileName: *req.FileName,
	}

	rpcResp, err := l.svcCtx.ContentRpc.Uploads(l.ctx, rpcReq)
	if err != nil {
		return nil, err
	}

	return &types.ContentUploadsCredentialsRes{
		ObjectKey: rpcResp.ObjectKey,
		FormData: types.OssFormData{
			Host:             rpcResp.FormData.Host,
			Policy:           rpcResp.FormData.Policy,
			Signature:        rpcResp.FormData.Signature,
			SecurityToken:    rpcResp.FormData.SecurityToken,
			SignatureVersion: rpcResp.FormData.SignatureVersion,
			Credential:       rpcResp.FormData.Credential,
			Date:             rpcResp.FormData.Date,
			Key:              rpcResp.FormData.Key,
		},
		ExpiredAt: rpcResp.ExpiredAt,
	}, nil

}
