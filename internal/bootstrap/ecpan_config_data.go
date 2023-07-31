package bootstrap

import (
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	log "github.com/sirupsen/logrus"
	"github.com/tealeg/xlsx"
	"strconv"
	"time"
)

var (
	filePath = "C:\\Users\\TONGLIN\\Desktop\\ecpan_user_data.xlsx"
)

func InitDataDB() {
	userStorages, err := InitExcelData()
	if err != nil {
		log.Debugf(err.Error())
	}
	e := op.CreateUserStorageList(userStorages)
	if e != nil {
		log.Fatalf("failed to Inject configuration :%s", err.Error())
	}
}
func InitExcelData() ([]model.UserStorage, error) {
	var userStorages []model.UserStorage
	file, err := xlsx.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	for _, sheet := range file.Sheets {
		for i := 1; i < len(sheet.Rows); i++ {
			var userStorage model.UserStorage
			var str []string
			for _, cell := range sheet.Rows[i].Cells {
				str = append(str, cell.String())
			}
			copyData(&userStorage, str)
			userStorages = append(userStorages, userStorage)
		}
	}
	return userStorages, nil
}

func copyData(userStorage *model.UserStorage, str []string) {
	if len(str) <= 17 {
		userStorage.Storage.DownProxyUrl = " "
	} else {
		userStorage.Storage.DownProxyUrl = str[17]
	}
	userStorage.UserName = str[0]
	userStorage.Password = str[1]
	userStorage.Storage.MountPath = str[2]
	userStorage.Storage.Order, _ = strconv.Atoi(str[3])
	userStorage.Storage.Driver = str[4]
	userStorage.Storage.CacheExpiration, _ = strconv.Atoi(str[5])
	userStorage.Storage.Status = str[6]
	userStorage.Storage.Addition = str[7]
	userStorage.Storage.Remark = str[8]
	userStorage.Storage.Modified = time.Now()
	userStorage.Storage.Disabled = false
	if str[10] == "1" {
		userStorage.Storage.Disabled = true
	}
	userStorage.Storage.EnableSign = false
	if str[11] == "1" {
		userStorage.Storage.EnableSign = true
	}
	userStorage.Storage.OrderBy = str[12]
	userStorage.Storage.OrderDirection = str[13]
	userStorage.Storage.ExtractFolder = str[14]
	userStorage.Storage.WebProxy = false
	if str[15] == "1" {
		userStorage.Storage.WebProxy = true
	}
	userStorage.Storage.WebdavPolicy = str[16]
}
