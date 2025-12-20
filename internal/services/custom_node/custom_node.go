package custom_node

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/node_health"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type CustomNodeService struct {
	db *gorm.DB
}

func NewCustomNodeService() *CustomNodeService {
	db := database.GetDB()
	return &CustomNodeService{
		db: db,
	}
}

func (s *CustomNodeService) GetDB() *gorm.DB {
	return s.db
}

func (s *CustomNodeService) CreateCustomNode(req *CreateNodeRequest) (*models.CustomNode, error) {
	customNode := models.CustomNode{
		Name:             req.Name,
		DisplayName:      req.DisplayName,
		Protocol:         req.Protocol,
		Config:           req.Config,
		Status:           "active",
		IsActive:         true,
		ExpireTime:       req.ExpireTime,
		FollowUserExpire: req.FollowUserExpire,
	}

	// 从配置中提取端口和域名
	var nodeConfig models.NodeConfig
	if err := json.Unmarshal([]byte(req.Config), &nodeConfig); err == nil {
		customNode.Port = nodeConfig.Port
		customNode.Domain = nodeConfig.Server
	}

	if err := s.db.Create(&customNode).Error; err != nil {
		return nil, fmt.Errorf("创建专线节点失败: %w", err)
	}

	return &customNode, nil
}

func (s *CustomNodeService) CreateCustomNodeFromLink(link string, preview bool) (*models.CustomNode, error) {
	proxyNode, err := config_update.ParseNodeLink(link)
	if err != nil {
		return nil, fmt.Errorf("节点链接解析失败: %w", err)
	}

	nodeConfig := models.NodeConfig{
		Type:       proxyNode.Type,
		Server:     proxyNode.Server,
		Port:       proxyNode.Port,
		Network:    proxyNode.Network,
		UUID:       proxyNode.UUID,
		Password:   proxyNode.Password,
		Encryption: proxyNode.Cipher,
		Security:   "tls",
	}

	if proxyNode.Options != nil {
		if v, ok := proxyNode.Options["servername"].(string); ok {
			nodeConfig.SNI = v
		}
		if v, ok := proxyNode.Options["fp"].(string); ok {
			nodeConfig.Fingerprint = v
		}
		if v, ok := proxyNode.Options["flow"].(string); ok {
			nodeConfig.Flow = v
		}
		if v, ok := proxyNode.Options["pbk"].(string); ok {
			nodeConfig.PublicKey = v
		}
		if v, ok := proxyNode.Options["sid"].(string); ok {
			nodeConfig.ShortID = v
		}
		if v, ok := proxyNode.Options["alpn"].(string); ok {
			nodeConfig.ALPN = v
		}
		if v, ok := proxyNode.Options["host"].(string); ok {
			nodeConfig.Host = v
		}
		if v, ok := proxyNode.Options["path"].(string); ok {
			nodeConfig.Path = v
		}
		if v, ok := proxyNode.Options["serviceName"].(string); ok {
			nodeConfig.ServiceName = v
		}
		if v, ok := proxyNode.Options["padding"].(bool); ok {
			nodeConfig.Padding = v
		}
		if v, ok := proxyNode.Options["congestion_control"].(string); ok {
			nodeConfig.CongestionControl = v
		}
		if v, ok := proxyNode.Options["udp_relay_mode"].(string); ok {
			nodeConfig.UDPRelayMode = v
		}
		if v, ok := proxyNode.Options["allowInsecure"].(bool); ok {
			nodeConfig.SkipCertVerify = v
		}
	}

	configBytes, _ := json.Marshal(nodeConfig)
	configStr := string(configBytes)

	customNode := models.CustomNode{
		Name:     proxyNode.Name,
		Protocol: proxyNode.Type,
		Port:     proxyNode.Port,
		Domain:   proxyNode.Server,
		Config:   configStr,
		Status:   "active",
		IsActive: true,
	}

	if preview {
		return &customNode, nil
	}

	// 检查是否已存在相同节点
	var existingNode models.CustomNode
	if err := s.db.Where("name = ? AND protocol = ? AND port = ?", proxyNode.Name, proxyNode.Type, proxyNode.Port).First(&existingNode).Error; err == nil {
		return nil, fmt.Errorf("节点已存在")
	}

	if err := s.db.Create(&customNode).Error; err != nil {
		return nil, fmt.Errorf("创建专线节点失败: %w", err)
	}

	return &customNode, nil
}

type CreateNodeRequest struct {
	Name             string     `json:"name"`
	DisplayName      string     `json:"display_name"`
	Protocol         string     `json:"protocol"`
	Config           string     `json:"config"`
	ExpireTime       *time.Time `json:"expire_time,omitempty"`
	FollowUserExpire bool       `json:"follow_user_expire"`
}

func (s *CustomNodeService) AssignNodeToUser(userID uint, customNodeID uint, subscriptionType string, expiresAt *time.Time) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}
	var node models.CustomNode
	if err := s.db.First(&node, customNodeID).Error; err != nil {
		return err
	}

	// 更新用户专线订阅设置
	if subscriptionType != "" {
		user.SpecialNodeSubscriptionType = subscriptionType
	}
	if expiresAt != nil {
		user.SpecialNodeExpiresAt = sql.NullTime{Time: *expiresAt, Valid: true}
	}

	if err := s.db.Save(&user).Error; err != nil {
		return err
	}

	// 检查是否已分配
	var existing models.UserCustomNode
	if err := s.db.Where("user_id = ? AND custom_node_id = ?", userID, customNodeID).First(&existing).Error; err == nil {
		return nil // 已分配
	}

	userCustomNode := models.UserCustomNode{UserID: userID, CustomNodeID: customNodeID}
	return s.db.Create(&userCustomNode).Error
}

func (s *CustomNodeService) UnassignNodeFromUser(userID uint, customNodeID uint) error {
	return s.db.Where("user_id = ? AND custom_node_id = ?", userID, customNodeID).Delete(&models.UserCustomNode{}).Error
}

func (s *CustomNodeService) BatchDeleteCustomNodes(nodeIDs []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("custom_node_id IN ?", nodeIDs).Delete(&models.UserCustomNode{})
		return tx.Delete(&models.CustomNode{}, nodeIDs).Error
	})
}

func (s *CustomNodeService) BatchAssignNodesToUsers(nodeIDs []uint, userIDs []uint, subscriptionType string, expiresAt *time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, uid := range userIDs {
			// 更新用户专线订阅设置
			var user models.User
			if tx.First(&user, uid).Error == nil {
				if subscriptionType != "" {
					user.SpecialNodeSubscriptionType = subscriptionType
				}
				if expiresAt != nil {
					user.SpecialNodeExpiresAt = sql.NullTime{Time: *expiresAt, Valid: true}
				}
				tx.Save(&user)
			}

			for _, nid := range nodeIDs {
				var exist models.UserCustomNode
				if tx.Where("user_id = ? AND custom_node_id = ?", uid, nid).First(&exist).Error != nil {
					tx.Create(&models.UserCustomNode{UserID: uid, CustomNodeID: nid})
				}
			}
		}
		return nil
	})
}

func (s *CustomNodeService) GetUserCustomNodes(userID uint) ([]models.CustomNode, error) {
	var nodes []models.CustomNode
	err := s.db.Joins("JOIN user_custom_nodes ON user_custom_nodes.custom_node_id = custom_nodes.id").
		Where("user_custom_nodes.user_id = ? AND custom_nodes.is_active = ?", userID, true).
		Find(&nodes).Error
	return nodes, err
}

func (s *CustomNodeService) GetCustomNodeUsers(nodeID uint) ([]models.User, error) {
	var users []models.User
	err := s.db.Joins("JOIN user_custom_nodes ON user_custom_nodes.user_id = users.id").
		Where("user_custom_nodes.custom_node_id = ?", nodeID).
		Find(&users).Error
	return users, err
}

func (s *CustomNodeService) CheckNodeExpiry(customNodeID uint, userExpireTime *time.Time) (bool, error) {
	var node models.CustomNode
	if err := s.db.First(&node, customNodeID).Error; err != nil {
		return false, err
	}
	if node.FollowUserExpire && userExpireTime != nil {
		return time.Now().After(*userExpireTime), nil
	}
	if node.ExpireTime != nil {
		return time.Now().After(*node.ExpireTime), nil
	}
	return false, nil
}

func (s *CustomNodeService) GenerateNodeLink(customNode *models.CustomNode) (string, error) {
	var nodeConfig models.NodeConfig
	if err := json.Unmarshal([]byte(customNode.Config), &nodeConfig); err != nil {
		return "", fmt.Errorf("解析节点配置失败: %w", err)
	}
	displayName := customNode.DisplayName
	if displayName == "" {
		displayName = "专线定制-" + customNode.Name
	}
	proxyNode := &config_update.ProxyNode{
		Name:     displayName,
		Type:     nodeConfig.Type,
		Server:   nodeConfig.Server,
		Port:     nodeConfig.Port,
		UUID:     nodeConfig.UUID,
		Password: nodeConfig.Password,
		Network:  nodeConfig.Network,
		Cipher:   nodeConfig.Encryption,
		TLS:      nodeConfig.Security == "tls",
		Options:  make(map[string]interface{}),
	}

	if nodeConfig.Security == "tls" {
		proxyNode.Options["security"] = "tls"
		if nodeConfig.SNI != "" {
			proxyNode.Options["servername"] = nodeConfig.SNI
		}
		if nodeConfig.Fingerprint != "" {
			proxyNode.Options["fp"] = nodeConfig.Fingerprint
		}
		if nodeConfig.ALPN != "" {
			proxyNode.Options["alpn"] = nodeConfig.ALPN
		}
		if nodeConfig.SkipCertVerify {
			proxyNode.Options["allowInsecure"] = true
		}
	}
	if nodeConfig.Flow != "" {
		proxyNode.Options["flow"] = nodeConfig.Flow
	}
	if nodeConfig.PublicKey != "" {
		proxyNode.Options["pbk"] = nodeConfig.PublicKey
	}
	if nodeConfig.ShortID != "" {
		proxyNode.Options["sid"] = nodeConfig.ShortID
	}
	if nodeConfig.Host != "" {
		proxyNode.Options["host"] = nodeConfig.Host
	}
	if nodeConfig.Path != "" {
		proxyNode.Options["path"] = nodeConfig.Path
	}
	if nodeConfig.ServiceName != "" {
		proxyNode.Options["serviceName"] = nodeConfig.ServiceName
	}
	if nodeConfig.Padding {
		proxyNode.Options["padding"] = true
	}
	if nodeConfig.CongestionControl != "" {
		proxyNode.Options["congestion_control"] = nodeConfig.CongestionControl
	}
	if nodeConfig.UDPRelayMode != "" {
		proxyNode.Options["udp_relay_mode"] = nodeConfig.UDPRelayMode
	}

	configService := config_update.NewConfigUpdateService()
	return configService.ProxyNodeToLink(proxyNode), nil
}

// 节点健康测试
func (s *CustomNodeService) TestCustomNode(customNodeID uint) (*node_health.TestResult, error) {
	var node models.CustomNode
	s.db.First(&node, customNodeID)
	var cfg models.NodeConfig
	json.Unmarshal([]byte(node.Config), &cfg)

	address := net.JoinHostPort(cfg.Server, strconv.Itoa(cfg.Port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)

	res := &node_health.TestResult{NodeID: customNodeID, TestedAt: time.Now(), Status: "online"}
	if err != nil {
		res.Status = "offline"
		res.Error = err.Error()
	} else {
		res.Latency = int(time.Since(start).Milliseconds())
		conn.Close()
	}
	s.db.Model(&node).Update("status", res.Status)
	return res, nil
}
