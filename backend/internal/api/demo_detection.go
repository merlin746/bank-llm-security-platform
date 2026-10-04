package api

import (
	"errors"
	"net"
	"regexp"
	"strings"
)

// demoDetection makes the active detector and any reduced capability explicit.
type demoDetection struct {
	Source   string `json:"source"`
	Mode     string `json:"mode"`
	Degraded bool   `json:"degraded"`
	Reason   string `json:"reason"`
}

var gatewayPromptRules = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`(?s)(忽略|忘记|无视).{0,30}?(之前|此前|先前|上面|系统|所有).{0,30}?(指令|规则|限制|提示)`), "覆盖已有指令或安全限制"},
	{regexp.MustCompile(`(?s)(泄露|透露|揭示|输出|显示|提供).{0,30}?(系统.{0,12}?(提示词|提示|指令)|内部.{0,12}?(安全策略|策略配置|配置)|安全策略配置)`), "泄露内部策略或系统指令"},
	{regexp.MustCompile(`(?s)(绕过|跳过|规避|无视).{0,20}?(权限|授权|访问控制|安全校验|安全检测)`), "绕过权限或安全校验"},
	{regexp.MustCompile(`(?s)(解除|取消|关闭|禁用).{0,20}?(安全限制|安全规则|安全检查|内容过滤)|无限制.{0,12}?(回答|输出)|不受.{0,12}?(约束|限制).{0,12}?(回答|助手|模型)`), "解除安全限制或无约束输出"},
	{regexp.MustCompile(`(?s)(?i)(你(现在)?是|扮演|充当).{0,16}?(\bDAN\b|不受(任何)?限制|无限制|无约束)`), "切换到无约束角色"},
	{regexp.MustCompile(`(?s)(泄露|窃取|盗取|套取).{0,24}?(密码|口令|密钥|凭证|客户数据|客户资料|身份证|银行卡号)|(?i)\bextract\b.{0,60}?\b(password|secret|key|credential)\b`), "窃取敏感信息"},
	{regexp.MustCompile(`(?s)(?i)\b(ignore|forget|disregard)\b.{0,80}?\b(previous|prior|above|system|all)\b.{0,80}?\b(instructions?|rules?|prompts?|restrictions?)\b`), "覆盖已有指令或安全限制"},
	{regexp.MustCompile(`(?s)(?i)\b(reveal|show|leak|print|disclose)\b.{0,80}?\b(system prompt|internal (configuration|policy|instructions))\b`), "泄露内部策略或系统指令"},
	{regexp.MustCompile(`(?s)(?i)\b(bypass|ignore|disable)\b.{0,80}?\b(permissions?|authorization|access control|security checks?|safety (filters?|rules?))\b`), "绕过权限或安全校验"},
	{regexp.MustCompile(`(?s)(?i)\b(you are|act as|pretend to be)\b.{0,30}?\b(dan|unrestricted|uncensored)\b|\bdo anything now\b|\bjailbreak\b`), "切换到无约束角色或越狱模式"},
}

var defensivePromptPrefix = regexp.MustCompile(`(?s)不要|不得|禁止|不应|不能|避免|防止|防范`)
var promptSentenceBoundary = regexp.MustCompile(`(?s)[，。！？；,;.!?\n]`)
var promptIntentChange = regexp.MustCompile(`(?s)但是|但|然而|不过|却|然后|改为|现在|请`)

func isDefensivePrompt(text string, matchStart int) bool {
	prefix := []rune(text[:matchStart])
	if len(prefix) > 24 {
		prefix = prefix[len(prefix)-24:]
	}
	parts := promptSentenceBoundary.Split(string(prefix), -1)
	clause := parts[len(parts)-1]
	guards := defensivePromptPrefix.FindAllStringIndex(clause, -1)
	if len(guards) == 0 {
		return false
	}
	tail := clause[guards[len(guards)-1][1]:]
	return len([]rune(tail)) <= 20 && !promptIntentChange.MatchString(tail)
}

func gatewayPromptRisk(prompt string) (bool, string) {
	text := prompt
	for _, rule := range gatewayPromptRules {
		for _, match := range rule.pattern.FindAllStringIndex(text, -1) {
			if !isDefensivePrompt(text, match[0]) {
				return true, rule.reason
			}
		}
	}
	return false, "未命中网关注入/越狱规则"
}

func (h *Handler) detectDemoPrompt(prompt string) (bool, string, map[string]interface{}, demoDetection) {
	result, err := h.callAI("/api/prompt/detect", map[string]interface{}{"text": prompt})
	detection := demoDetection{Source: "ai-service", Mode: "unknown"}
	if err == nil {
		detection.Mode = strOr(result["detection_mode"])
		if detection.Mode != "rules" && detection.Mode != "model" {
			switch strOr(result["layer"]) {
			case "rule", "rules":
				detection.Mode = "rules"
			case "model":
				detection.Mode = "model"
			default:
				err = errors.New("AI 服务未提供有效检测模式")
			}
		}
	}
	blocked := false
	reason := "未检测到注入/越狱意图"
	extra := map[string]interface{}{"riskScore": 12, "riskType": "normal"}
	if err != nil {
		blocked, reason = gatewayPromptRisk(prompt)
		detection = demoDetection{Source: "gateway-rules", Mode: "rules", Degraded: true, Reason: "AI 服务不可用，已回退网关规则检测"}
		if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
			detection.Reason = "AI 服务检测超时，已回退网关规则检测"
		} else if strings.HasPrefix(err.Error(), "AI 服务") {
			detection.Reason = err.Error() + "，已回退网关规则检测"
		}
	} else {
		blocked = result["is_attack"].(bool)
		if value := strOr(result["reason"]); value != "" {
			reason = value
		}
		if confidence, ok := result["confidence"].(float64); ok {
			extra["confidence"] = confidence
		}
		var hasDegraded bool
		detection.Degraded, hasDegraded = result["degraded"].(bool)
		detection.Reason = strOr(result["degradation_reason"])
		if !hasDegraded && detection.Mode == "rules" {
			detection.Degraded = true
			detection.Reason = "AI 服务仅返回规则判定，未提供模型可用状态"
		} else if detection.Degraded && detection.Reason == "" {
			detection.Reason = "AI 服务已降级为规则检测"
		}
	}
	if blocked {
		extra["riskScore"] = 90
		extra["riskType"] = "jailbreak"
		reason = "检测到越狱/注入意图：" + reason
	}
	if detection.Degraded {
		reason += "；" + detection.Reason
	}
	extra["detection"] = detection
	return blocked, reason, extra, detection
}
