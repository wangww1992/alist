package _Ecpan

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
)

type Ecpan struct {
	model.Storage
	Addition
}

func (d *Ecpan) Config() driver.Config {
	return config
}

func (d *Ecpan) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *Ecpan) Init(ctx context.Context) error {
	_, err := d.post("/service/disk/search.do?func=disk:userInfo", base.Json{}, nil)
	return err
}

func (d *Ecpan) Drop(ctx context.Context) error {
	return nil
}

func (d *Ecpan) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	return d.getFiles(dir.GetID())
}

func (d *Ecpan) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	u, err := d.getLink(file.GetID())
	if err != nil {
		return nil, err
	}
	return &model.Link{URL: u}, nil
}

func (d *Ecpan) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) error {
	currentDir := parentDir.GetID()
	if currentDir == d.RootFolderID {
		currentDir = ""
	}
	data := base.Json{
		"comeFrom":   10,
		"creatorUsn": d.Account,
		"diskType":   1,
		"fileName":   dirName,
		"groupId":    "",
		"parentId":   currentDir,
	}
	pathname := "/service/common/file.do?func=common:createFolder"
	_, err := d.post(pathname, data, nil)
	return err
}

func (d *Ecpan) Move(ctx context.Context, srcObj, dstDir model.Obj) error {
	return d.doMoveOrCopy(srcObj, dstDir, false)
}

func (d *Ecpan) Rename(ctx context.Context, srcObj model.Obj, newName string) error {
	var data base.Json
	var pathname string
	if srcObj.IsDir() {
		data = base.Json{
			"appFileId":   srcObj.GetID(),
			"comeFrom":    10,
			"creatorUsn":  d.Account,
			"diskType":    1,
			"fileType":    1,
			"newFileName": newName,
		}
	} else {
		data = base.Json{
			"appFileId":   srcObj.GetID(),
			"comeFrom":    10,
			"creatorUsn":  d.Account,
			"diskType":    1,
			"fileType":    2,
			"newFileName": newName,
		}
	}
	pathname = "/service/common/file.do?func=common:rename"
	_, err := d.post(pathname, data, nil)
	return err
}

func (d *Ecpan) Copy(ctx context.Context, srcObj, dstDir model.Obj) error {
	return d.doMoveOrCopy(srcObj, dstDir, false)
}

func (d *Ecpan) Remove(ctx context.Context, obj model.Obj) error {
	data := base.Json{
		"comeFrom":   10,
		"diskType":   1,
		"creatorUsn": d.Account,
		"fileIdList": []base.Json{
			{
				"appFileId": obj.GetID(),
			},
		},
		"isComplete": 0,
	}

	pathname := "/service/common/file.do?func=common:newDeleteFile"
	var resp SimpleResp
	_, err := d.post(pathname, data, &resp)
	if err != nil {
		return err
	}
	data = base.Json{
		"taskId": resp.Var,
	}
	pathname = "/service/common/file.do?func=common:OpFileProgress"
	_, err = d.post(pathname, data, nil)
	return err
}

func (d *Ecpan) Put(ctx context.Context, dstDir model.Obj, stream model.FileStreamer, up driver.UpdateProgress) error {
	tempFile, err := utils.CreateTempFile(stream.GetReadCloser())
	if err != nil {
		return err
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	md5, err := GetFileMd5(tempFile)
	_, err = tempFile.Seek(0, io.SeekStart)

	parentFolder := base.Json{}
	parentId := "-1"
	if dstDir.GetID() != d.RootFolderID {
		parentId = dstDir.GetID()
		parentFolder = base.Json{
			"appFileId": dstDir.GetID(),
		}
	}
	if err != nil {
		return err
	}
	data := base.Json{
		"model":          0,
		"newFlag":        1,
		"parentId":       parentId,
		"diskType":       1,
		"comeFrom":       10,
		"parentFolder":   parentFolder,
		"discussContent": "",
		"fileMd5":        md5,
		"fileName":       stream.GetName(),
		"fileSize":       stream.GetSize(),
		"uploadType":     3,
	}
	pathname := "/service/common/file.do?func=common:upload"
	var preResp PreUploadResp
	_, err = d.post(pathname, data, &preResp)
	if err != nil {
		if err.Error() == "DFS_118" {
			return nil
		}
		return err
	}
	// Progress

	partSize, remainSize, start, end, _ := getPartInfo(stream.GetSize(), 0)
	partCount := (stream.GetSize() + partSize - 1) / partSize
	partNum := 0
	hasNextPart := true
	for hasNextPart {
		partSize, remainSize, start, end, hasNextPart = getPartInfo(stream.GetSize(), int64(partNum))
		partNum++
		if !hasNextPart {
			partSize = remainSize
		}

		sliceData := make([]byte, partSize)
		_, err = io.ReadFull(tempFile, sliceData)
		if err != nil {
			return err
		}
		sliceFile, err := CreateTempFileFromBytes(sliceData)
		if err != nil {
			return err
		}
		defer func() {
			_ = sliceFile.Close()
			_ = os.Remove(sliceFile.Name())
		}()
		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		bodyWriter.WriteField("parentId", dstDir.GetID())
		bodyWriter.WriteField("fileName", stream.GetName())
		bodyWriter.WriteField("fileSize", fmt.Sprintf("%d", stream.GetSize()))
		bodyWriter.WriteField("uploadId", preResp.Data.UploadId)
		bodyWriter.WriteField("dfsFileId", preResp.Data.DfsFileId)
		bodyWriter.WriteField("partNum", fmt.Sprintf("%d", partNum))
		bodyWriter.WriteField("partCount", fmt.Sprintf("%d", partCount))
		bodyWriter.WriteField("start", fmt.Sprintf("%d", start))
		bodyWriter.WriteField("range", fmt.Sprintf("%d-%d", start, end))
		bodyWriter.WriteField("fileMd5", md5)
		bodyWriter.WriteField("uploadType", "3")
		bodyWriter.WriteField("model", "0")
		bodyWriter.WriteField("diskType", "1")
		bodyWriter.WriteField("comeFrom", "10")

		fileWriter, err := bodyWriter.CreateFormFile("file", sliceFile.Name())
		if err != nil {
			return err
		}
		fh, err := os.Open(sliceFile.Name())
		if err != nil {
			return err
		}
		defer fh.Close()
		_, err = io.Copy(fileWriter, fh)
		if err != nil {
			return err
		}
		bodyWriter.Close()

		request, err := http.NewRequest("POST", preResp.Data.FastUploadUrl, bodyBuf)
		if err != nil {
			return err
		}
		request.Header.Set("Cookie", "sid="+d.SessionId)
		request.Header.Set("Content-Type", bodyWriter.FormDataContentType())
		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	}
	data = base.Json{
		"comeFrom": 10,
		"parentId": dstDir.GetID(),
		"model":    0,
		"diskType": 1,
		"parentFolder": base.Json{
			"appFileId": dstDir.GetID(),
		},
		"fileMd5":    md5,
		"uploadId":   preResp.Data.UploadId,
		"dfsFileId":  preResp.Data.DfsFileId,
		"partCount":  partCount,
		"fileSize":   stream.GetSize(),
		"fileName":   stream.GetName(),
		"uploadType": 3,
	}
	pathname = "/service/common/file.do?func=common:completeUpload"
	_, err = d.post(pathname, data, nil)
	if err != nil {
		return err
	}
	return nil
}

var _ driver.Driver = (*Ecpan)(nil)
