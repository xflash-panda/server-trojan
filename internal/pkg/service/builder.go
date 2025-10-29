package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	cProtocol "github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/task"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/inbound"
	"github.com/xtls/xray-core/features/stats"
	"github.com/xtls/xray-core/proxy"

	api "github.com/xflash-panda/server-client/pkg"
)

type Config struct {
	NodeID                 int
	FetchUsersInterval     time.Duration
	ReportTrafficsInterval time.Duration
	HeartbeatInterval      time.Duration
	Cert                   *CertConfig
	ExtConfPath            string
	ServerHost             string
	ServerPort             int
	ListenAddr             string
}

// APIClient 定义了与API服务器交互所需的方法接口
type APIClient interface {
	Users(string, api.NodeType) (*[]api.User, error)
	Submit(string, api.NodeType, []*api.UserTraffic) error
	Heartbeat(string, api.NodeType, string) error
}

type Builder struct {
	instance                      *core.Instance
	config                        *Config
	nodeInfo                      *api.TrojanConfig
	inboundTag                    string
	userList                      *[]api.User
	registerId                    string
	apiClient                     APIClient
	fetchUsersMonitorPeriodic     *task.Periodic
	reportTrafficsMonitorPeriodic *task.Periodic
	heartbeatMonitorPeriodic      *task.Periodic
}

// New return a builder service with default parameters.
func New(inboundTag string, instance *core.Instance, config *Config, nodeInfo *api.TrojanConfig, registerId string,
	apiClient APIClient,
) *Builder {
	builder := &Builder{
		inboundTag: inboundTag,
		instance:   instance,
		config:     config,
		nodeInfo:   nodeInfo,
		registerId: registerId,
		apiClient:  apiClient,
	}
	return builder
}

// addUsers
func (b *Builder) addUsers(users []*cProtocol.User, tag string) error {
	inboundManager := b.instance.GetFeature(inbound.ManagerType()).(inbound.Manager)
	handler, err := inboundManager.GetHandler(context.Background(), tag)
	if err != nil {
		return fmt.Errorf("no such inbound tag: %s", err)
	}
	inboundInstance, ok := handler.(proxy.GetInbound)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.GetInbound", tag)
	}

	userManager, ok := inboundInstance.GetInbound().(proxy.UserManager)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.UserManager", err)
	}
	for _, item := range users {
		mUser, err := item.ToMemoryUser()
		if err != nil {
			return err
		}
		err = userManager.AddUser(context.Background(), mUser)
		if err != nil {
			return err
		}
	}
	return nil
}

// addNewUser
func (b *Builder) addNewUser(userInfo []api.User) (err error) {
	users := buildUser(b.inboundTag, userInfo)
	err = b.addUsers(users, b.inboundTag)
	if err != nil {
		return err
	}
	log.Infof("Added %d new users", len(userInfo))
	return nil
}

// Start implement the Start() function of the service interface
func (b *Builder) Start() error {
	// Update user
	userList, err := b.apiClient.Users(b.registerId, api.Trojan)
	if err != nil {
		return err
	}
	err = b.addNewUser(*userList)
	if err != nil {
		return err
	}

	b.userList = userList

	b.fetchUsersMonitorPeriodic = &task.Periodic{
		Interval: b.config.FetchUsersInterval,
		Execute:  b.fetchUsersMonitor,
	}
	b.reportTrafficsMonitorPeriodic = &task.Periodic{
		Interval: b.config.ReportTrafficsInterval,
		Execute:  b.reportTrafficsMonitor,
	}
	log.Infoln("Start monitoring for user acquisition")
	err = b.fetchUsersMonitorPeriodic.Start()
	if err != nil {
		return fmt.Errorf("fetch users monitor periodic, start erorr:%s", err)
	}
	log.Infoln("Start traffic reporting monitoring")
	err = b.reportTrafficsMonitorPeriodic.Start()
	if err != nil {
		return fmt.Errorf("start traffic monitor periodic, start erorr:%s", err)
	}

	if b.config.HeartbeatInterval > 0 {
		b.heartbeatMonitorPeriodic = &task.Periodic{
			Interval: b.config.HeartbeatInterval,
			Execute:  b.heartbeatMonitor,
		}
		log.Infoln("Start heartbeat monitoring")
		err = b.heartbeatMonitorPeriodic.Start()
		if err != nil {
			return fmt.Errorf("start heartbeat monitor periodic, start erorr:%s", err)
		}
	}
	return nil
}

// Close implement the Close() function of the service interface
func (b *Builder) Close() error {
	if b.fetchUsersMonitorPeriodic != nil {
		err := b.fetchUsersMonitorPeriodic.Close()
		if err != nil {
			return fmt.Errorf("fetch users monitor periodic close failed: %s", err)
		}
	}

	if b.reportTrafficsMonitorPeriodic != nil {
		err := b.reportTrafficsMonitorPeriodic.Close()
		if err != nil {
			return fmt.Errorf("report  traffics monitor periodic close failed: %s", err)
		}
	}

	if b.heartbeatMonitorPeriodic != nil {
		err := b.heartbeatMonitorPeriodic.Close()
		if err != nil {
			return fmt.Errorf("heartbeat monitor periodic close failed: %s", err)
		}
	}
	return nil
}

// getTraffic
func (b *Builder) getTraffic(email string) (up int64, down int64, count int64) {
	upName := "user>>>" + email + ">>>traffic>>>uplink"
	downName := "user>>>" + email + ">>>traffic>>>downlink"
	countName := "user>>>" + email + ">>>request>>>count"
	statsManager := b.instance.GetFeature(stats.ManagerType()).(stats.Manager)
	upCounter := statsManager.GetCounter(upName)
	downCounter := statsManager.GetCounter(downName)
	countCounter := statsManager.GetCounter(countName)
	if upCounter != nil {
		up = upCounter.Value()
		upCounter.Set(0)
	}
	if downCounter != nil {
		down = downCounter.Value()
		downCounter.Set(0)
	}
	if countCounter != nil {
		count = countCounter.Value()
		countCounter.Set(0)
	}

	return up, down, count
}

// removeUsers
func (b *Builder) removeUsers(users []string, tag string) error {
	inboundManager := b.instance.GetFeature(inbound.ManagerType()).(inbound.Manager)
	handler, err := inboundManager.GetHandler(context.Background(), tag)
	if err != nil {
		return fmt.Errorf("no such inbound tag: %s", err)
	}
	inboundInstance, ok := handler.(proxy.GetInbound)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.GetInbound", tag)
	}

	userManager, ok := inboundInstance.GetInbound().(proxy.UserManager)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.UserManager", err)
	}
	for _, email := range users {
		err = userManager.RemoveUser(context.Background(), email)
		if err != nil {
			return err
		}
	}
	return nil
}

// nodeInfoMonitor
func (b *Builder) fetchUsersMonitor() (err error) {
	// Update User
	newUserList, err := b.apiClient.Users(b.registerId, api.Trojan)
	if err != nil {
		// 数据未修改，忽略
		if errors.Is(err, api.ErrorUserNotModified) {
			return nil
		}
		// 区分客户端错误和服务端错误
		var apiError *api.APIError
		if errors.As(err, &apiError) {
			if apiError.IsServerError() {
				// 服务端错误，打印日志
				log.Errorln("server error when fetching users", err)
				return nil
			}
		}
		// 其他未知错误，打印日志
		log.Errorln(err)
		return nil
	}
	deleted, added := b.compareUserList(newUserList)
	if len(deleted) > 0 {
		deletedEmail := make([]string, len(deleted))
		for i, u := range deleted {
			deletedEmail[i] = buildUserEmail(b.inboundTag, u.ID, u.UUID)
		}
		err := b.removeUsers(deletedEmail, b.inboundTag)
		if err != nil {
			log.Errorln(err)
			return nil
		}
	}
	if len(added) > 0 {
		err = b.addNewUser(added)
		if err != nil {
			log.Errorln(err)
			return nil
		}

	}
	log.Infof("%d user deleted, %d user added", len(deleted), len(added))

	b.userList = newUserList
	return nil
}

// userInfoMonitor
func (b *Builder) reportTrafficsMonitor() (err error) {
	// Get User traffic
	userTraffic := make([]*api.UserTraffic, 0)
	for _, user := range *b.userList {
		email := buildUserEmail(b.inboundTag, user.ID, user.UUID)
		up, down, count := b.getTraffic(email)
		if up > 0 || down > 0 || count > 0 {
			userTraffic = append(userTraffic, &api.UserTraffic{
				UID:      user.ID,
				Upload:   uint64(up),
				Download: uint64(down),
				Count:    uint64(count),
			})
		}
	}
	log.Infof("%d user traffic needs to be reported", len(userTraffic))
	if len(userTraffic) > 0 {
		err = b.apiClient.Submit(b.registerId, api.Trojan, userTraffic)
		if err != nil {
			var apiError *api.APIError
			if errors.As(err, &apiError) {
				if apiError.IsServerError() {
					log.Errorln("server error when submitting traffic", err)
					return nil
				}
			}
		}
	}

	return nil
}

// compareUserList
func (b *Builder) compareUserList(newUsers *[]api.User) (deleted, added []api.User) {
	// 使用map来标记旧用户列表中的每个用户
	userMap := make(map[api.User]bool)

	// 标记旧用户列表中所有用户为已删除（暂时）
	for _, user := range *b.userList {
		userMap[user] = true
	}

	// 遍历新用户列表
	for _, newUser := range *newUsers {
		if userMap[newUser] {
			// 如果当前用户在旧列表中，标记为未删除（即用户仍在列表中）
			userMap[newUser] = false
		} else {
			// 如果用户不在旧列表中，那么它是一个新增用户
			added = append(added, newUser)
		}
	}

	// 任何在userMap中仍标记为true的用户都是被删除的
	for user, isDeleted := range userMap {
		if isDeleted {
			deleted = append(deleted, user)
		}
	}

	return deleted, added
}

// heartbeatMonitor
func (b *Builder) heartbeatMonitor() (err error) {
	err = b.apiClient.Heartbeat(b.registerId, api.Trojan, "")
	if err != nil {
		var apiError *api.APIError
		if errors.As(err, &apiError) {
			if apiError.IsServerError() {
				log.Errorln("server error when sending heartbeat", err)
				return nil
			}
		}
	}
	return nil
}
