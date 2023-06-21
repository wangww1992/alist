package _Ecpan

import (
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/op"
)

type Addition struct {
	Account string `json:"account" required:"true"`
	// Cookie  string `json:"cookie" type:"text" required:"true"`
	SessionId string `json:"sid" type:"text" required:"true"`
	driver.RootID
}

var config = driver.Config{
	Name:      "Ecpan",
	LocalSort: true,
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &Ecpan{}
	})
}
