package enum

import (
	"fmt"
	"strings"
)

// AgentType 对应 Java AgentTypeEnum 的一个枚举项
type AgentType struct {
	AgentName string
	Desc      string
}

// 对应 Java AgentTypeEnum（desc 为原文，1:1 复刻）
var (
	ReActPlanningAgent = AgentType{"reActPlanningAgent", "任务规划智能体"}
	BrowserAgent       = AgentType{"BrowserAgent", "浏览器Agent可以进行通用浏览器操作，例如通过网站查询到需要的信息或是进行指定的网页操作"}
	TableAgent         = AgentType{"TableAgent", "此Agent专职用于绘制表格，只能基于上下文中已有的数据进行绘制，无法查询额外信息"}
	ChartAgent         = AgentType{"ChartAgent", "此Agent专职用于绘制统计图，只能基于上下文中已有的数据进行绘制，无法查询额外信息"}
	HtmlDocAgent       = AgentType{"HtmlDocAgent", "此Agent用于生成各类网页内容，只能基于上下文中已有的数据进行生成，无法查询额外信息；可作为生成一般内容时的默认Agent"}
	AMAPAgent          = AgentType{"AMAPAgent", "此Agent包含完整的地图工具集，可用于路线规划、结构化地址转换为经纬度坐标等地理信息操作，返回文字或多媒体链接的结果"}
)

var agentTypes = []AgentType{
	ReActPlanningAgent, BrowserAgent, TableAgent, ChartAgent, HtmlDocAgent, AMAPAgent,
}

// AgentNameOf 对应 AgentTypeEnum.agentNameOf()，未找到返回 nil
func AgentNameOf(agentName string) *AgentType {
	for i := range agentTypes {
		if agentTypes[i].AgentName == agentName {
			return &agentTypes[i]
		}
	}
	return nil
}

// AgentData 生成「可选Agent」列表，用于填入 promptPlanningUserTask 的 {agentData}
func AgentData() string {
	var sb strings.Builder
	for i, a := range agentTypes {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(fmt.Sprintf("- %s: %s", a.AgentName, a.Desc))
	}
	return sb.String()
}
