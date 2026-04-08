package file

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/KubeOperator/kubepi/internal/api/v1/session"
	fileModel "github.com/KubeOperator/kubepi/internal/model/v1/file"
	v1System "github.com/KubeOperator/kubepi/internal/model/v1/system"
	"github.com/KubeOperator/kubepi/internal/service/v1/common"
	"github.com/KubeOperator/kubepi/internal/service/v1/file"
	v1SystemService "github.com/KubeOperator/kubepi/internal/service/v1/system"
	"github.com/KubeOperator/kubepi/pkg/util/requestip"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

type Handler struct {
	fileService file.Service
}

func NewHandler() *Handler {
	return &Handler{
		fileService: file.NewService(),
	}
}

func (h *Handler) ListFiles() iris.Handler {
	return func(ctx *context.Context) {
		var req fileModel.Request
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		res, err := h.fileService.ListFiles(req)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		ctx.Values().Set("data", res)
	}
}

func (h *Handler) CreateFolder() iris.Handler {
	return func(ctx *context.Context) {
		var req fileModel.Request
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		req.Commands = []string{"mkdir", req.Path}
		if _, err := h.fileService.ExecNewCommand(req); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
	}
}

func (h *Handler) CreateFile() iris.Handler {
	return func(ctx *context.Context) {
		var req fileModel.Request
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		command := "echo '" + req.Content + "' >> " + req.Path
		req.Commands = []string{"sh", "-c", command}
		if _, err := h.fileService.ExecNewCommand(req); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
	}
}

func (h *Handler) UpdateFile() iris.Handler {
	return func(ctx *context.Context) {
		var req fileModel.Request
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		if err := h.fileService.EditFile(req); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
	}
}

func (h *Handler) OpenFile() iris.Handler {
	return func(ctx *context.Context) {
		var req fileModel.Request
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		res, err := h.fileService.CatFile(req)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		ctx.Values().Set("data", string(res))
	}
}

func (h *Handler) ReNameFile() iris.Handler {
	return func(ctx *context.Context) {
		var req fileModel.Request
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		if req.OldPath == "" {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", "file or path is not exist")
			return
		}
		req.Commands = []string{"mv", req.OldPath, req.Path}
		_, err := h.fileService.ExecNewCommand(req)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
	}
}

func (h *Handler) RmFolder() iris.Handler {
	return func(ctx *context.Context) {
		var req fileModel.Request
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		req.Commands = []string{"rm", req.Path}
		if _, err := h.fileService.ExecNewCommand(req); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
	}
}

func (h *Handler) DownloadFolder() iris.Handler {
	return func(ctx *context.Context) {

		var req fileModel.Request
		req.Path = ctx.URLParam("path")
		req.Namespace = ctx.URLParam("namespace")
		req.Cluster = ctx.URLParam("cluster")
		req.PodName = ctx.URLParam("podName")
		req.ContainerName = ctx.URLParam("containerName")

		file, err := h.fileService.DownloadFolder(req)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			savePodExportAuditLog(ctx, "folder", req, iris.StatusInternalServerError, false, err.Error())
			return
		}
		filename := path.Base(file)
		err = ctx.SendFile(file, filename)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			savePodExportAuditLog(ctx, "folder", req, iris.StatusInternalServerError, false, err.Error())
			return
		}
		os.RemoveAll(file)
		savePodExportAuditLog(ctx, "folder", req, iris.StatusOK, true, "")
	}
}
func (h *Handler) DownloadFile() iris.Handler {
	return func(ctx *context.Context) {

		var req fileModel.Request
		req.Path = ctx.URLParam("path")
		req.Namespace = ctx.URLParam("namespace")
		req.Cluster = ctx.URLParam("cluster")
		req.PodName = ctx.URLParam("podName")
		req.ContainerName = ctx.URLParam("containerName")

		file, err := h.fileService.DownloadFile(req)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			savePodExportAuditLog(ctx, "file", req, iris.StatusInternalServerError, false, err.Error())
			return
		}
		filename := path.Base(file)
		err = ctx.SendFile(file, filename)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			savePodExportAuditLog(ctx, "file", req, iris.StatusInternalServerError, false, err.Error())
			return
		}
		os.RemoveAll(file)
		savePodExportAuditLog(ctx, "file", req, iris.StatusOK, true, "")
	}
}

func savePodExportAuditLog(ctx *context.Context, exportKind string, req fileModel.Request, statusCode int, success bool, errMsg string) {
	p := ctx.Values().Get("profile")
	profile, ok := p.(session.UserProfile)
	if !ok || profile.Name == "" {
		return
	}
	op := "export_file"
	if exportKind == "folder" {
		op = "export_folder"
	}
	record := fmt.Sprintf("数据导出(%s)：cluster=%s namespace=%s pod=%s container=%s path=%s",
		exportKind, req.Cluster, req.Namespace, req.PodName, req.ContainerName, req.Path)
	if !success && errMsg != "" {
		msg := errMsg
		if len(msg) > 280 {
			msg = msg[:280] + "…"
		}
		record = record + "；失败: " + msg
	}
	logItem := v1System.AuditLog{
		Operator:            profile.Name,
		Cluster:             req.Cluster,
		HttpMethod:          "get",
		RequestPath:         ctx.Request().URL.Path,
		Resource:            "pod",
		Operation:           op,
		OperationDomain:     "pod_files",
		SpecificInformation: fmt.Sprintf("[%s/%s] %s", req.Cluster, req.Namespace, req.PodName),
		OperationRecord:     record,
		ClientIp:            requestip.FromRequest(ctx.Request()),
		UserAgent:           ctx.GetHeader("User-Agent"),
		StatusCode:          statusCode,
		Success:             success,
	}
	svc := v1SystemService.NewService()
	go svc.CreateAuditLog(&logItem, common.DBOptions{})
}

func (h *Handler) UploadFile() iris.Handler {
	return func(ctx *context.Context) {
		var req fileModel.Request
		req.Path = ctx.URLParam("path")
		req.Namespace = ctx.URLParam("namespace")
		req.Cluster = ctx.URLParam("cluster")
		req.PodName = ctx.URLParam("podName")
		req.ContainerName = ctx.URLParam("containerName")

		srcPath := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().UnixNano()))
		err := saveTarFile(ctx, srcPath)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}

		req.FilePath = srcPath
		err = h.fileService.UploadFile(req)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
	}
}

func saveTarFile(ctx *context.Context, srcPath string) error {
	maxSize := ctx.Application().ConfigurationReadOnly().GetPostMaxMemory()
	err := ctx.Request().ParseMultipartForm(maxSize)
	if err != nil {
		return err
	}
	form := ctx.Request().MultipartForm
	files := form.File["files"]
	if len(files) == 0 {
		return errors.New("files is null")
	}
	fw, err := os.Create(srcPath)
	defer fw.Close()
	if err != nil {
		return err
	}
	tw := tar.NewWriter(fw)
	defer tw.Close()
	for _, f := range files {
		hdr := &tar.Header{
			Name: f.Filename,
			Mode: 0644,
			Size: f.Size,
		}
		err = tw.WriteHeader(hdr)
		if err != nil {
			return err
		}
		_f, err := f.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, _f)
		if err != nil {
			return err
		}
	}
	return nil
}

func Install(parent iris.Party) {
	handler := NewHandler()
	sp := parent.Party("/pod")
	sp.Post("/files", handler.ListFiles())
	sp.Post("/folder/create", handler.CreateFolder())
	sp.Post("/folder/delete", handler.RmFolder())
	sp.Post("/files/create", handler.CreateFile())
	sp.Post("/files/open", handler.OpenFile())
	sp.Post("/files/rename", handler.ReNameFile())
	sp.Post("/files/upload", handler.UploadFile())
	sp.Post("/files/update", handler.UpdateFile())
	sp.Get("/files/download/folder", handler.DownloadFolder())
	sp.Get("/files/download/file", handler.DownloadFile())
}
