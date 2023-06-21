package _Ecpan

type BaseResp struct {
	Code      string `json:"code"`
	ErrorCode string `json:"errorCode"`
	Summary   string `json:"summary"`
}

type SimpleResp struct {
	Code      string `json:"code"`
	ErrorCode string `json:"errorCode"`
	Summary   string `json:"summary"`
	Var       string `json:"var"`
}

type Catalog struct {
	CatalogID   string `json:"catalogID"`
	CatalogName string `json:"catalogName"`
	UpdateTime  string `json:"updateTime"`
}

type GetDiskResp struct {
	BaseResp
	Data struct {
		ParentPath  string `json:"parentPath"`
		PageCount   int64  `json:"pageCount"`
		RecordCount int64  `json:"recordCount"`
		ResultList  []struct {
			AppFileId  string `json:"appFileId"`
			CreateDate string `json:"createDate"`
			ModifyDate string `json:"modifyDate"`
			UploadTime string `json:"uploadTime"`
			FileName   string `json:"fileName"`
			FilePath   string `json:"filePath"`
			FileSize   int64  `json:"fileSize"`
			FileSort   int64  `json:"fileSort"`
			FileType   int64  `json:"fileType"`
			FolderType int64  `json:"folderType"`
		} `json:"resultList"`
	} `json:"var"`
}

type PreUploadResp struct {
	BaseResp
	Data struct {
		DfsFileId     string `json:"dfsFileId"`
		FastUploadUrl string `json:"fastUploadUrl"`
		UploadId      string `json:"uploadId"`
	} `json:"var"`
}

type CloudContent struct {
	ContentID      string `json:"contentID"`
	ContentName    string `json:"contentName"`
	ContentSize    int64  `json:"contentSize"`
	LastUpdateTime string `json:"lastUpdateTime"`
	ThumbnailURL   string `json:"thumbnailURL"`
}

type CloudCatalog struct {
	CatalogID      string `json:"catalogID"`
	CatalogName    string `json:"catalogName"`
	LastUpdateTime string `json:"lastUpdateTime"`
}

type QueryContentListResp struct {
	BaseResp
	Data struct {
		Result struct {
			ResultCode string `json:"resultCode"`
			ResultDesc string `json:"resultDesc"`
		} `json:"result"`
		Path             string         `json:"path"`
		CloudContentList []CloudContent `json:"cloudContentList"`
		CloudCatalogList []CloudCatalog `json:"cloudCatalogList"`
		TotalCount       int            `json:"totalCount"`
		RecallContent    interface{}    `json:"recallContent"`
	} `json:"data"`
}
