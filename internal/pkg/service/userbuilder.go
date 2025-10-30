package service

import (
	"fmt"

	cProtocol "github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/trojan"

	api "github.com/xflash-panda/server-client/pkg"
)

func buildUser(tag string, userInfo []api.User) (users []*cProtocol.User) {
	users = make([]*cProtocol.User, len(userInfo))
	for i, user := range userInfo {
		trojanAccount := &trojan.Account{
			Password: user.UUID,
		}
		email := buildUserEmail(tag, user.ID, user.UUID)
		users[i] = &cProtocol.User{
			Level:   0,
			Email:   email,
			Account: serial.ToTypedMessage(trojanAccount),
		}
	}
	return users
}

func buildUserEmail(tag string, id int, uuid string) string {
	return fmt.Sprintf("%s|%d|%s", tag, id, uuid)
}
