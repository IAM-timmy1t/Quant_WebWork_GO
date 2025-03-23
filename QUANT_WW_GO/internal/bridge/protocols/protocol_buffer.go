// protocol_buffer.go - Protocol Buffer message formats and service definitions

package protocols

import (
	"fmt"
	"strings"
	"time"
)

// APIVersion represents the version of the protocol buffer messages
type APIVersion string

// API versions
const (
	APIVersionV1 APIVersion = "v1"
	APIVersionV2 APIVersion = "v2"
)

// MessageType represents the type of message
type MessageType string

// Message types
const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeEvent    MessageType = "event"
	MessageTypeError    MessageType = "error"
)

// ServiceType represents the type of service
type ServiceType string

// Service types
const (
	ServiceTypeAnalyzer  ServiceType = "analyzer"
	ServiceTypeRisk      ServiceType = "risk"
	ServiceTypeStorage   ServiceType = "storage"
	ServiceTypeMonitor   ServiceType = "monitor"
	ServiceTypeAPI       ServiceType = "api"
	ServiceTypeAuth      ServiceType = "auth"
	ServiceTypeDiscovery ServiceType = "discovery"
)

// ErrorCode represents standardized error codes
type ErrorCode string

// Standard error codes
const (
	ErrorCodeInvalidRequest      ErrorCode = "INVALID_REQUEST"
	ErrorCodeInternalError       ErrorCode = "INTERNAL_ERROR"
	ErrorCodeServiceUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeResourceNotFound    ErrorCode = "RESOURCE_NOT_FOUND"
	ErrorCodePermissionDenied    ErrorCode = "PERMISSION_DENIED"
	ErrorCodeUnauthenticated     ErrorCode = "UNAUTHENTICATED"
	ErrorCodeResourceExhausted   ErrorCode = "RESOURCE_EXHAUSTED"
	ErrorCodeDeadlineExceeded    ErrorCode = "DEADLINE_EXCEEDED"
	ErrorCodeAlreadyExists       ErrorCode = "ALREADY_EXISTS"
	ErrorCodeFailedPrecondition  ErrorCode = "FAILED_PRECONDITION"
	ErrorCodeAborted             ErrorCode = "ABORTED"
	ErrorCodeOutOfRange          ErrorCode = "OUT_OF_RANGE"
	ErrorCodeUnimplemented       ErrorCode = "UNIMPLEMENTED"
	ErrorCodeDataLoss            ErrorCode = "DATA_LOSS"
	ErrorCodeUnavailable         ErrorCode = "UNAVAILABLE"
)

// MessageHeader contains common fields for all messages
type MessageHeader struct {
	MessageID     string      `json:"message_id" protobuf:"1"`
	Version       APIVersion  `json:"version" protobuf:"2"`
	Type          MessageType `json:"type" protobuf:"3"`
	Timestamp     int64       `json:"timestamp" protobuf:"4"`
	CorrelationID string      `json:"correlation_id,omitempty" protobuf:"5"`
	Source        string      `json:"source,omitempty" protobuf:"6"`
	Destination   string      `json:"destination,omitempty" protobuf:"7"`
	TraceID       string      `json:"trace_id,omitempty" protobuf:"8"`
	ServiceName   string      `json:"service_name,omitempty" protobuf:"9"`
	ServiceType   ServiceType `json:"service_type,omitempty" protobuf:"10"`
}

// NewMessageHeader creates a new message header
func NewMessageHeader(msgType MessageType, version APIVersion, source string, destination string) MessageHeader {
	return MessageHeader{
		MessageID:     NewUUID(),
		Version:       version,
		Type:          msgType,
		Timestamp:     time.Now().UnixNano() / int64(time.Millisecond),
		CorrelationID: NewUUID(),
		Source:        source,
		Destination:   destination,
		TraceID:       NewUUID(),
	}
}

// NewUUID generates a new UUID v4
func NewUUID() string {
	// Simple UUID generation for example purposes
	// In a real implementation, you would use a proper UUID library
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

// Message represents a generic protocol buffer message
type Message struct {
	Header  MessageHeader   `json:"header" protobuf:"1"`
	Payload map[string]any  `json:"payload" protobuf:"2"`
	Error   *ErrorMessage   `json:"error,omitempty" protobuf:"3"`
	Meta    map[string]any  `json:"meta,omitempty" protobuf:"4"`
}

// ErrorMessage represents an error in a protocol buffer message
type ErrorMessage struct {
	Code    ErrorCode `json:"code" protobuf:"1"`
	Message string    `json:"message" protobuf:"2"`
	Details any       `json:"details,omitempty" protobuf:"3"`
}

// ServiceDefinition defines a service and its methods
type ServiceDefinition struct {
	Name        string             `json:"name" protobuf:"1"`
	Version     APIVersion         `json:"version" protobuf:"2"`
	Type        ServiceType        `json:"type" protobuf:"3"`
	Description string             `json:"description,omitempty" protobuf:"4"`
	Methods     []MethodDefinition `json:"methods" protobuf:"5"`
}

// MethodDefinition defines a service method
type MethodDefinition struct {
	Name        string               `json:"name" protobuf:"1"`
	Description string               `json:"description,omitempty" protobuf:"2"`
	InputType   string               `json:"input_type" protobuf:"3"`
	OutputType  string               `json:"output_type" protobuf:"4"`
	Type        MethodType           `json:"type" protobuf:"5"`
	Options     map[string]string    `json:"options,omitempty" protobuf:"6"`
	Deprecated  bool                 `json:"deprecated,omitempty" protobuf:"7"`
	Since       APIVersion           `json:"since,omitempty" protobuf:"8"`
	Params      []ParameterDefinition `json:"params,omitempty" protobuf:"9"`
}

// MethodType represents the type of method
type MethodType string

// Method types
const (
	MethodTypeUnary        MethodType = "unary"
	MethodTypeServerStream MethodType = "server_stream"
	MethodTypeClientStream MethodType = "client_stream"
	MethodTypeBidiStream   MethodType = "bidi_stream"
)

// ParameterDefinition defines a method parameter
type ParameterDefinition struct {
	Name        string `json:"name" protobuf:"1"`
	Type        string `json:"type" protobuf:"2"`
	Description string `json:"description,omitempty" protobuf:"3"`
	Required    bool   `json:"required,omitempty" protobuf:"4"`
	Default     any    `json:"default,omitempty" protobuf:"5"`
}

// TokenAnalysisService defines the token analyzer service
var TokenAnalysisService = ServiceDefinition{
	Name:        "TokenAnalyzer",
	Version:     APIVersionV1,
	Type:        ServiceTypeAnalyzer,
	Description: "Service for analyzing token properties and behavior",
	Methods: []MethodDefinition{
		{
			Name:        "AnalyzeToken",
			Description: "Analyzes a token by its contract address",
			InputType:   "AnalyzeTokenRequest",
			OutputType:  "AnalyzeTokenResponse",
			Type:        MethodTypeUnary,
			Params: []ParameterDefinition{
				{
					Name:        "token_address",
					Type:        "string",
					Description: "The token's contract address",
					Required:    true,
				},
				{
					Name:        "options",
					Type:        "map<string, any>",
					Description: "Analysis options",
					Required:    false,
				},
			},
		},
		{
			Name:        "BatchAnalyzeTokens",
			Description: "Analyzes multiple tokens in a single request",
			InputType:   "BatchAnalyzeTokensRequest",
			OutputType:  "BatchAnalyzeTokensResponse",
			Type:        MethodTypeUnary,
			Params: []ParameterDefinition{
				{
					Name:        "token_addresses",
					Type:        "list<string>",
					Description: "List of token contract addresses",
					Required:    true,
				},
				{
					Name:        "options",
					Type:        "map<string, any>",
					Description: "Analysis options",
					Required:    false,
				},
			},
		},
		{
			Name:        "StreamTokenUpdates",
			Description: "Streams real-time updates about a token",
			InputType:   "TokenUpdateRequest",
			OutputType:  "TokenUpdateResponse",
			Type:        MethodTypeServerStream,
			Params: []ParameterDefinition{
				{
					Name:        "token_address",
					Type:        "string",
					Description: "The token's contract address",
					Required:    true,
				},
				{
					Name:        "update_types",
					Type:        "list<string>",
					Description: "Types of updates to receive",
					Required:    false,
				},
			},
		},
	},
}

// RiskAnalysisService defines the risk analyzer service
var RiskAnalysisService = ServiceDefinition{
	Name:        "RiskAnalyzer",
	Version:     APIVersionV1,
	Type:        ServiceTypeRisk,
	Description: "Service for analyzing token risk profiles",
	Methods: []MethodDefinition{
		{
			Name:        "CalculateRisk",
			Description: "Calculates risk metrics for a token",
			InputType:   "CalculateRiskRequest",
			OutputType:  "CalculateRiskResponse",
			Type:        MethodTypeUnary,
			Params: []ParameterDefinition{
				{
					Name:        "token_address",
					Type:        "string",
					Description: "The token's contract address",
					Required:    true,
				},
				{
					Name:        "risk_factors",
					Type:        "list<string>",
					Description: "Risk factors to include",
					Required:    false,
				},
			},
		},
		{
			Name:        "CompareRisks",
			Description: "Compares risk profiles between multiple tokens",
			InputType:   "CompareRisksRequest",
			OutputType:  "CompareRisksResponse",
			Type:        MethodTypeUnary,
			Params: []ParameterDefinition{
				{
					Name:        "token_addresses",
					Type:        "list<string>",
					Description: "List of token contract addresses",
					Required:    true,
				},
				{
					Name:        "comparison_factors",
					Type:        "list<string>",
					Description: "Factors to compare",
					Required:    false,
				},
			},
		},
	},
}

// DiscoveryService defines the service discovery service
var DiscoveryService = ServiceDefinition{
	Name:        "Discovery",
	Version:     APIVersionV1,
	Type:        ServiceTypeDiscovery,
	Description: "Service for discovering and managing service instances",
	Methods: []MethodDefinition{
		{
			Name:        "RegisterService",
			Description: "Registers a service instance",
			InputType:   "RegisterServiceRequest",
			OutputType:  "RegisterServiceResponse",
			Type:        MethodTypeUnary,
			Params: []ParameterDefinition{
				{
					Name:        "service_info",
					Type:        "ServiceInfo",
					Description: "Service information",
					Required:    true,
				},
			},
		},
		{
			Name:        "DeregisterService",
			Description: "Deregisters a service instance",
			InputType:   "DeregisterServiceRequest",
			OutputType:  "DeregisterServiceResponse",
			Type:        MethodTypeUnary,
			Params: []ParameterDefinition{
				{
					Name:        "service_id",
					Type:        "string",
					Description: "Service ID",
					Required:    true,
				},
			},
		},
		{
			Name:        "DiscoverServices",
			Description: "Discovers services by type and version",
			InputType:   "DiscoverServicesRequest",
			OutputType:  "DiscoverServicesResponse",
			Type:        MethodTypeUnary,
			Params: []ParameterDefinition{
				{
					Name:        "service_type",
					Type:        "string",
					Description: "Type of service",
					Required:    false,
				},
				{
					Name:        "version",
					Type:        "string",
					Description: "Service version",
					Required:    false,
				},
			},
		},
		{
			Name:        "WatchServices",
			Description: "Watches for service changes",
			InputType:   "WatchServicesRequest",
			OutputType:  "WatchServicesResponse",
			Type:        MethodTypeServerStream,
			Params: []ParameterDefinition{
				{
					Name:        "service_type",
					Type:        "string",
					Description: "Type of service",
					Required:    false,
				},
			},
		},
	},
}

// === Specific Message Definitions ===

// ServiceInfo represents information about a service instance
type ServiceInfo struct {
	ID          string            `json:"id" protobuf:"1"`
	Name        string            `json:"name" protobuf:"2"`
	Version     APIVersion        `json:"version" protobuf:"3"`
	Type        ServiceType       `json:"type" protobuf:"4"`
	Address     string            `json:"address" protobuf:"5"`
	Port        int               `json:"port" protobuf:"6"`
	Metadata    map[string]string `json:"metadata,omitempty" protobuf:"7"`
	Status      string            `json:"status" protobuf:"8"`
	LastUpdated int64             `json:"last_updated" protobuf:"9"`
}

// TokenAnalysisResult represents the result of a token analysis
type TokenAnalysisResult struct {
	TokenAddress      string                  `json:"token_address" protobuf:"1"`
	TokenName         string                  `json:"token_name,omitempty" protobuf:"2"`
	TokenSymbol       string                  `json:"token_symbol,omitempty" protobuf:"3"`
	ContractType      string                  `json:"contract_type,omitempty" protobuf:"4"`
	StandardCompliance []string               `json:"standard_compliance,omitempty" protobuf:"5"`
	Findings          []TokenFinding          `json:"findings,omitempty" protobuf:"6"`
	Recommendations   []TokenRecommendation   `json:"recommendations,omitempty" protobuf:"7"`
	RiskProfile       *TokenRiskProfile       `json:"risk_profile,omitempty" protobuf:"8"`
	ContractInfo      *TokenContractInfo      `json:"contract_info,omitempty" protobuf:"9"`
	TransactionHistory *TokenTransactionHistory `json:"transaction_history,omitempty" protobuf:"10"`
	Analytics         map[string]any          `json:"analytics,omitempty" protobuf:"11"`
	Timestamp         int64                   `json:"timestamp" protobuf:"12"`
	AnalysisVersion   string                  `json:"analysis_version" protobuf:"13"`
}

// TokenFinding represents a finding from token analysis
type TokenFinding struct {
	ID          string      `json:"id" protobuf:"1"`
	Title       string      `json:"title" protobuf:"2"`
	Description string      `json:"description" protobuf:"3"`
	Severity    string      `json:"severity" protobuf:"4"`
	Category    string      `json:"category" protobuf:"5"`
	Location    string      `json:"location,omitempty" protobuf:"6"`
	Evidence    any         `json:"evidence,omitempty" protobuf:"7"`
	Confidence  float64     `json:"confidence,omitempty" protobuf:"8"`
	FirstFound  int64       `json:"first_found,omitempty" protobuf:"9"`
	LastFound   int64       `json:"last_found" protobuf:"10"`
	References  []string    `json:"references,omitempty" protobuf:"11"`
	IsFixed     bool        `json:"is_fixed,omitempty" protobuf:"12"`
}

// TokenRecommendation represents a recommendation for addressing findings
type TokenRecommendation struct {
	ID          string                   `json:"id" protobuf:"1"`
	Title       string                   `json:"title" protobuf:"2"`
	Description string                   `json:"description" protobuf:"3"`
	Priority    string                   `json:"priority" protobuf:"4"`
	RelatedFindingIDs []string           `json:"related_finding_ids,omitempty" protobuf:"5"`
	Actions     []TokenRecommendationAction `json:"actions,omitempty" protobuf:"6"`
}

// TokenRecommendationAction represents an action to address a recommendation
type TokenRecommendationAction struct {
	ID          string      `json:"id" protobuf:"1"`
	Title       string      `json:"title" protobuf:"2"`
	Description string      `json:"description" protobuf:"3"`
	ActionType  string      `json:"action_type" protobuf:"4"`
	Priority    string      `json:"priority" protobuf:"5"`
	CodeSnippet string      `json:"code_snippet,omitempty" protobuf:"6"`
}

// TokenRiskProfile represents a token's risk assessment
type TokenRiskProfile struct {
	OverallRiskScore   float64              `json:"overall_risk_score" protobuf:"1"`
	RiskLevel          string               `json:"risk_level" protobuf:"2"`
	RiskFactors        map[string]float64   `json:"risk_factors" protobuf:"3"`
	RiskCategories     map[string]RiskCategory `json:"risk_categories" protobuf:"4"`
	HistoricalRisk     []HistoricalRiskEntry `json:"historical_risk,omitempty" protobuf:"5"`
	ConfidenceScore    float64              `json:"confidence_score" protobuf:"6"`
	AnalysisTimestamp  int64                `json:"analysis_timestamp" protobuf:"7"`
}

// RiskCategory represents a category of risk assessment
type RiskCategory struct {
	Name        string              `json:"name" protobuf:"1"`
	Score       float64             `json:"score" protobuf:"2"`
	Level       string              `json:"level" protobuf:"3"`
	Factors     map[string]float64  `json:"factors" protobuf:"4"`
	Description string              `json:"description,omitempty" protobuf:"5"`
}

// HistoricalRiskEntry represents a historical risk assessment entry
type HistoricalRiskEntry struct {
	Timestamp    int64   `json:"timestamp" protobuf:"1"`
	RiskScore    float64 `json:"risk_score" protobuf:"2"`
	RiskLevel    string  `json:"risk_level" protobuf:"3"`
	ChangeReason string  `json:"change_reason,omitempty" protobuf:"4"`
}

// TokenContractInfo represents information about the token contract
type TokenContractInfo struct {
	Address           string            `json:"address" protobuf:"1"`
	Name              string            `json:"name,omitempty" protobuf:"2"`
	Symbol            string            `json:"symbol,omitempty" protobuf:"3"`
	Decimals          int               `json:"decimals,omitempty" protobuf:"4"`
	TotalSupply       string            `json:"total_supply,omitempty" protobuf:"5"`
	CreationTimestamp int64             `json:"creation_timestamp,omitempty" protobuf:"6"`
	Creator           string            `json:"creator,omitempty" protobuf:"7"`
	BlockNumber       int64             `json:"block_number,omitempty" protobuf:"8"`
	BytecodeHash      string            `json:"bytecode_hash,omitempty" protobuf:"9"`
	SourceCodeVerified bool              `json:"source_code_verified,omitempty" protobuf:"10"`
	CompilerVersion   string            `json:"compiler_version,omitempty" protobuf:"11"`
	OptimizationUsed  bool              `json:"optimization_used,omitempty" protobuf:"12"`
	Runs              int               `json:"runs,omitempty" protobuf:"13"`
	ConstructorArgs   string            `json:"constructor_args,omitempty" protobuf:"14"`
	EVMVersion        string            `json:"evm_version,omitempty" protobuf:"15"`
	Library           string            `json:"library,omitempty" protobuf:"16"`
	LicenseType       string            `json:"license_type,omitempty" protobuf:"17"`
	SwarmSource       string            `json:"swarm_source,omitempty" protobuf:"18"`
	ABI               string            `json:"abi,omitempty" protobuf:"19"`
	Implementation    string            `json:"implementation,omitempty" protobuf:"20"`
}

// TokenTransactionHistory represents transaction history data for a token
type TokenTransactionHistory struct {
	TotalTransactions   int64                `json:"total_transactions" protobuf:"1"`
	UniqueAddresses     int64                `json:"unique_addresses" protobuf:"2"`
	TransactionsPerDay  []DailyTransactions  `json:"transactions_per_day,omitempty" protobuf:"3"`
	TopTransactions     []TokenTransaction   `json:"top_transactions,omitempty" protobuf:"4"`
	VolumeStats         TokenVolumeStats     `json:"volume_stats,omitempty" protobuf:"5"`
	HolderDistribution  []HolderBucket       `json:"holder_distribution,omitempty" protobuf:"6"`
}

// DailyTransactions represents daily transaction data
type DailyTransactions struct {
	Date         string  `json:"date" protobuf:"1"`
	Count        int64   `json:"count" protobuf:"2"`
	Volume       string  `json:"volume" protobuf:"3"`
	AvgPrice     string  `json:"avg_price,omitempty" protobuf:"4"`
	UniqueWallets int64   `json:"unique_wallets,omitempty" protobuf:"5"`
}

// TokenTransaction represents a significant token transaction
type TokenTransaction struct {
	TransactionHash  string  `json:"transaction_hash" protobuf:"1"`
	Timestamp        int64   `json:"timestamp" protobuf:"2"`
	From             string  `json:"from" protobuf:"3"`
	To               string  `json:"to" protobuf:"4"`
	Value            string  `json:"value" protobuf:"5"`
	BlockNumber      int64   `json:"block_number" protobuf:"6"`
	TransactionIndex int     `json:"transaction_index" protobuf:"7"`
	Type             string  `json:"type,omitempty" protobuf:"8"`
	GasUsed          int64   `json:"gas_used,omitempty" protobuf:"9"`
	GasPrice         string  `json:"gas_price,omitempty" protobuf:"10"`
}

// TokenVolumeStats represents volume statistics for a token
type TokenVolumeStats struct {
	TotalVolume     string  `json:"total_volume" protobuf:"1"`
	AverageVolume   string  `json:"average_volume" protobuf:"2"`
	HighestVolume   string  `json:"highest_volume" protobuf:"3"`
	HighestVolumeDate string `json:"highest_volume_date" protobuf:"4"`
	LowestVolume    string  `json:"lowest_volume" protobuf:"5"`
	LowestVolumeDate string `json:"lowest_volume_date" protobuf:"6"`
}

// HolderBucket represents a distribution bucket for token holders
type HolderBucket struct {
	Range       string  `json:"range" protobuf:"1"`
	Count       int64   `json:"count" protobuf:"2"`
	Percentage  float64 `json:"percentage" protobuf:"3"`
	TotalTokens string  `json:"total_tokens" protobuf:"4"`
}

// ServiceManifest represents a collection of service definitions
type ServiceManifest struct {
	Services []ServiceDefinition `json:"services" protobuf:"1"`
	Version  string              `json:"version" protobuf:"2"`
	Generated int64              `json:"generated" protobuf:"3"`
}

// DefaultServiceManifest returns the default service manifest
func DefaultServiceManifest() ServiceManifest {
	return ServiceManifest{
		Services: []ServiceDefinition{
			TokenAnalysisService,
			RiskAnalysisService,
			DiscoveryService,
		},
		Version:   string(APIVersionV1),
		Generated: time.Now().Unix(),
	}
}

// GetMethodByName returns a method definition by its name
func (s *ServiceDefinition) GetMethodByName(name string) *MethodDefinition {
	for i, method := range s.Methods {
		if method.Name == name || strings.EqualFold(method.Name, name) {
			return &s.Methods[i]
		}
	}
	return nil
}

// GetServiceByName returns a service definition by its name
func (m *ServiceManifest) GetServiceByName(name string) *ServiceDefinition {
	for i, service := range m.Services {
		if service.Name == name || strings.EqualFold(service.Name, name) {
			return &m.Services[i]
		}
	}
	return nil
}

// GetServiceByType returns a service definition by its type
func (m *ServiceManifest) GetServiceByType(serviceType ServiceType) *ServiceDefinition {
	for i, service := range m.Services {
		if service.Type == serviceType {
			return &m.Services[i]
		}
	}
	return nil
}

// ValidateMessage validates a message against its schema
func ValidateMessage(message interface{}, schema interface{}) error {
	// This is a placeholder for a real validation implementation
	// In a real implementation, you would use reflection or a schema validation library
	return nil
}

// GenerateProto generates a .proto file definition from a service definition
func (s *ServiceDefinition) GenerateProto() string {
	// This is a simplified implementation
	sb := strings.Builder{}
	
	sb.WriteString(fmt.Sprintf("syntax = \"proto3\";\n\n"))
	sb.WriteString(fmt.Sprintf("package %s;\n\n", strings.ToLower(string(s.Type))))
	sb.WriteString(fmt.Sprintf("option go_package = \"github.com/quant-webworks/go/internal/bridge/protocols/%s\";\n\n", strings.ToLower(string(s.Type))))
	
	sb.WriteString(fmt.Sprintf("// %s\n", s.Description))
	sb.WriteString(fmt.Sprintf("service %s {\n", s.Name))
	
	for _, method := range s.Methods {
		sb.WriteString(fmt.Sprintf("  // %s\n", method.Description))
		
		if method.Deprecated {
			sb.WriteString("  option deprecated = true;\n")
		}
		
		var requestStream, responseStream string
		if method.Type == MethodTypeClientStream || method.Type == MethodTypeBidiStream {
			requestStream = "stream "
		}
		if method.Type == MethodTypeServerStream || method.Type == MethodTypeBidiStream {
			responseStream = "stream "
		}
		
		sb.WriteString(fmt.Sprintf("  rpc %s(%s%s) returns (%s%s);\n\n", 
			method.Name, requestStream, method.InputType, responseStream, method.OutputType))
	}
	
	sb.WriteString("}\n\n")
	
	// Add message definitions
	// This would be more complex in a real implementation,
	// generating message types based on the parameters and return types
	
	return sb.String()
}

// ConvertToProtobufMessage converts a Go struct to a protocol buffer message
func ConvertToProtobufMessage(data interface{}) (*Message, error) {
	// This is a placeholder for a real implementation
	// In a real implementation, you would use reflection or a protocol buffer library
	return &Message{
		Header: MessageHeader{
			MessageID: NewUUID(),
			Version:   APIVersionV1,
			Type:      MessageTypeResponse,
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		},
		Payload: map[string]any{
			"data": data,
		},
	}, nil
}
