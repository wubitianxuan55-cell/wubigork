# 灵犀 Go 类型名清单（静态抽取，全量）

- 源: `C:\Users\wubi\AppData\Roaming\WPS 灵犀\app_server_5.24.0\service\lingxi.exe`
- 去重 (包, 类型) **15984** 对 / 包 **1365** 个
- 口径：从类型串表（`*pkg.Type`、`[]*pkg.Type` 等）抽取；只反映**出现过的类型名**，不含字段

## 包 × 类型数（前 40）

| 包 | 类型数 |
|---|---|
| `cos` | 797 |
| `v5` | 530 |
| `prompt` | 416 |
| `s3` | 342 |
| `conn` | 291 |
| `obs` | 289 |
| `cmdable` | 286 |
| `lib` | 282 |
| `oss` | 248 |
| `common` | 224 |
| `api` | 198 |
| `core` | 175 |
| `llm` | 156 |
| `etcdserverpb` | 156 |
| `windows` | 150 |
| `libc` | 148 |
| `protobuf` | 122 |
| `protocol` | 119 |
| `mcp` | 118 |
| `publicapi` | 115 |
| `v1` | 104 |
| `exec` | 98 |
| `memory` | 97 |
| `redis` | 95 |
| `v3` | 93 |
| `zap` | 85 |
| `syntax` | 83 |
| `metric` | 82 |
| `proto` | 80 |
| `tools` | 77 |
| `trace` | 76 |
| `clientv3` | 67 |
| `v8` | 67 |
| `serverapi` | 66 |
| `minio` | 63 |
| `ast` | 61 |
| `agent` | 61 |
| `util` | 60 |
| `ncoder` | 59 |
| `types` | 58 |

## 业务相关包的类型全量

### `core`（175）

`AccountBinding`、`AccountClientResolver`、`AccountCredentialStore`、`AccountIdentityFetcher`、`AccountLifecycleController`、`AccountLifecycleError`、`AccountLifecycleFailure`、`AccountLifecyclePolicy`、`AccountModule`、`AccountRegistry`、`AccountRuntime`、`AgentConfig`、`AgentFactory`、`AgentRegistry`、`ApplyClientTracingConfig`、`ApplyFaultInjectionGate`、`AuthConfig`、`AuthCredentialStore`、`AvailableMemorySampler`、`AvailableMemorySamplerFunc`、`BackendAPIConfig`、`BackendConfig`、`BashAuthProvider`、`BillingRunFactory`、`CLIConnectorCheckProvider`、`CacheExpireSnipConfig`、`ClientTracingConfigPending`、`CompactConfig`、`Config`、`ConfigManager`、`ConfigReadError`、`ConfigSaver`、`ConfigWriteError`、`Connection`、`ConnectionManager`、`ConnectorManageProvider`、`ConnectorResolveProvider`、`ConversationStoreOwner`、`CustomModelConfig`、`CustomModelStore`、`DWConfig`、`DebugConfig`、`DecodeEvent`、`DefaultAuthPath`、`DefaultConfig`、`DefaultConfigPath`、`DefaultDataDir`、`DeliveryStats`、`ExecutionLayer`、`ExpertRoleSelection`、`ExpertTransition`、`FaultInjectionConfig`、`FaultInjectionGateEnabled`、`GatewayIdentity`、`GatewayIdentityForClientType`、`Generation`、`IMHistoryCompactor`、`IMMemoryCompactConfig`、`IMMemoryCompactModelConfig`、`IdentityResolver`、`InitClientTracing`、`LLMConfig`、`LLMRequestDumpConfig`、`LoadConfig`、`MemoryConfig`、`MemoryRunBridge`、`MemoryRunRequest`、`MessageProcessor`、`MetricsConfig`、`ModelCatalog`、`ModelDescriptor`、`ModelRef`、`ModelUpdate`、`NetworkConfig`、`NewAccountLifecycleError`、`NewAccountModule`、`NewAccountRegistry`、`NewAnswerAccumulator`、`NewAuthCredentialStore`、`NewBillingRunFactory`、`NewCLIConnectorCheckProvider`、`NewConfigManager`、`NewConnectionManager`、`NewExecutionLayer`、`NewIMHistoryCompactor`、`NewMemoryRunBridge`、`NewMessageProcessor`、`NewModelCatalog`、`NewModelCatalogWithClientType`、`NewModelCatalogWithStoreAndClientType`、`NewPythonWorkerStartMetricsObserver`、`NewPythonWorkerSupervisor`、`NewRequestAccessPolicy`、`NewRunManager`、`NewRuntimeSessionRegistry`、`NewServerModelCatalog`、`NewSessionAgentRunner`、`NewWorkerMetricsObserver`、`PendingRequest`、`PersistenceBinding`、`PersistenceControl`、`PythonWorkerSupervisor`、`PythonWorkerSupervisorConfig`、`PythonWorkerSupervisorEvent`、`PythonWorkerSupervisorObserver`、`PythonWorkerSupervisorObserverFunc`、`PythonWorkerSupervisorTicker`、`RemoteLogConfig`、`RequestAccessPolicy`、`ResolveLogDir`、`Run`、`RunCancellationSource`、`RunCreditsClient`、`RunEvent`、`RunExecutionConfig`、`RunExecutionConfigFromContext`、`RunHandle`、`RunIDMismatchError`、`RunIDResolver`、`RunManager`、`RunStatus`、`RuntimeAPIConfig`、`RuntimeAuthRefreshProvider`、`RuntimeSessionRegistry`、`SameAuthIdentity`、`SandboxConfig`、`SaveConfig`、`ScopeKey`、`ServerConfig`、`ServerModelCatalog`、`ServerRequestDeliveryLease`、`Session`、`SessionAgentProfile`、`SessionAgentRunner`、`SessionCacheExpireSnip`、`SessionMemory`、`SessionMemoryItem`、`SessionProject`、`SessionRunConfig`、`SessionSkill`、`SessionSkills`、`SessionStartRequest`、`SessionTitleService`、`SessionWikiPageMeta`、`SessionWikiSnapshot`、`SessionWorkspace`、`SetupFromConfigWithOptions`、`SetupOption`、`SetupResult`、`ShutdownClientTracing`、`ShutdownGate`、`SkillListContextProvider`、`SortSessionSkillsOfficialFirst`、`StartRunFunc`、`TimerTaskProvider`、`TracingConfig`、`TracingEnvelope`、`TracingServerConfig`、`ValidateConfig`、`WPSCLIBrokerRootRegistrar`

### `agent`（61）

`APIFormat`、`AddedKey`、`Agent`、`BackendChatModel`、`BackendClient`、`BackendRequestDumpCleaner`、`BackendRequestDumper`、`ConnectorInfo`、`ConstraintExtension`、`DirectChatModel`、`DirectModelConfig`、`ExtendedAgent`、`HTTPWikiPageStore`、`JSONLBackendRequestDumper`、`Key`、`LLMConfig`、`LLMMode`、`LocalContextManager`、`LocalCredential`、`LocalModelConfiger`、`LocalOperator`、`LocalParentIDProvider`、`LocalProjectInsightStore`、`LocalSessionContext`、`LocalWikiPageStore`、`New`、`NewAnthropicChatModel`、`NewAnthropicClient`、`NewAuthManagedCredential`、`NewBackendChatModel`、`NewBackendClient`、`NewClient`、`NewDirectChatModel`、`NewDirectClient`、`NewHTTPWikiPageStore`、`NewJSONLBackendRequestDumper`、`NewLocalContextManager`、`NewLocalCredential`、`NewLocalCredentialWithLanguage`、`NewLocalCredentialWithLanguageAndQueryParams`、`NewLocalModelConfiger`、`NewLocalOperator`、`NewLocalParentIDProvider`、`NewLocalProjectInsightStore`、`NewLocalSessionContext`、`NewLocalSessionContextWithMessageIDGenerator`、`NewLocalWikiPageStore`、`NewResponsesChatModel`、`NewResponsesClient`、`NewTelemetryAnthropicClient`、`NewTelemetryClient`、`NewTelemetryDirectClient`、`NormalizeAPIFormat`、`ResponsesChatModel`、`ResponsesClient`、`ResponsesStream`、`RuntimeAuthRefreshFunc`、`SetBackendChatHTTPClient`、`SignatureFlags`、`SupportedAPIFormats`、`WrapTelemetryChatModel`

### `session`（18）

`AIOfficeFileInfo`、`AIOfficeUserInputContent`、`AgentInput`、`AssumeRoleTokenProviderNotSetError`、`CoworkAgentInput`、`CoworkFileInfo`、`CredentialRequiresARNError`、`CredentialsProviderOptions`、`Must`、`NewSession`、`NewSessionWithOptions`、`Options`、`Session`、`SharedConfigAssumeRoleError`、`SharedConfigLoadError`、`SharedConfigProfileNotExistsError`、`SharedConfigState`、`WpsArgs`

### `run`（1）

`WithoutCancel`

### `skill`（4）

`Description`、`Location`、`Name`、`Source`

### `connector`（4）

`Description`、`Kind`、`Name`、`Type`

### `mcp`（118）

`Annotations`、`AudioContent`、`CallToolParams`、`CallToolParamsRaw`、`CallToolResult`、`CancelledParams`、`Client`、`ClientCapabilities`、`ClientOptions`、`ClientRequest`、`ClientSession`、`ClientSessionOptions`、`CloseSSEStreamArgs`、`CompleteContext`、`CompleteParams`、`CompleteParamsArgument`、`CompleteReference`、`CompleteResult`、`CompletionCapabilities`、`CompletionResultDetails`、`Connection`、`Content`、`CreateMessageParams`、`CreateMessageResult`、`CreateMessageWithToolsParams`、`CreateMessageWithToolsResult`、`ElicitParams`、`ElicitResult`、`ElicitationCapabilities`、`ElicitationCompleteParams`、`EmbeddedResource`、`Event`、`FormElicitationCapabilities`、`GetPromptParams`、`GetPromptResult`、`IOTransport`、`Icon`、`IconTheme`、`ImageContent`、`Implementation`、`InitializeParams`、`InitializeResult`、`InitializedParams`、`ListPromptsParams`、`ListPromptsResult`、`ListResourceTemplatesParams`、`ListResourceTemplatesResult`、`ListResourcesParams`、`ListResourcesResult`、`ListRootsParams`、`ListRootsResult`、`ListToolsParams`、`ListToolsResult`、`LoggingCapabilities`、`LoggingLevel`、`LoggingMessageParams`、`Meta`、`MethodHandler`、`Middleware`、`ModelHint`、`ModelPreferences`、`NewClient`、`Params`、`PingParams`、`ProgressNotificationParams`、`Prompt`、`PromptArgument`、`PromptCapabilities`、`PromptHandler`、`PromptListChangedParams`、`PromptMessage`、`ReadResourceParams`、`ReadResourceResult`、`Request`、`RequestExtra`、`Resource`、`ResourceCapabilities`、`ResourceContents`、`ResourceHandler`、`ResourceLink`、`ResourceListChangedParams`、`ResourceNotFoundError`、`ResourceTemplate`、`ResourceUpdatedNotificationParams`、`Result`、`Role`、`Root`、`RootCapabilities`、`RootsListChangedParams`、`SamplingCapabilities`、`SamplingContextCapabilities`、`SamplingMessage`、`SamplingMessageV2`、`SamplingToolsCapabilities`、`SchemaCache`、`Server`、`ServerCapabilities`、`ServerOptions`、`ServerRequest`、`ServerSession`、`ServerSessionOptions`、`ServerSessionState`、`Session`、`SetLoggingLevelParams`、`StreamableClientTransport`、`SubscribeParams`、`TextContent`、`Tool`、`ToolAnnotations`、`ToolCapabilities`、`ToolChoice`、`ToolHandler`、`ToolListChangedParams`、`ToolResultContent`、`ToolUseContent`、`Transport`、`URLElicitationCapabilities`、`UnsubscribeParams`

### `sandbox`（1）

`Spawn`

### `auth`（17）

`AuthDotJson`、`AuthManager`、`AuthMode`、`BceCredentials`、`BceV1Signer`、`EffectiveSnapshot`、`EnvWPSSidIsSet`、`NewAuthManager`、`NewBceCredentials`、`NewRuntimeAuthManager`、`OAuthHandler`、`RuntimeRefreshOutcome`、`SignOptions`、`Signer`、`StorageMode`、`TokenInfo`、`WPSSidCredential`

### `config`（30）

`Branch`、`ClientConfig`、`Comment`、`Config`、`Decoder`、`Defaults`、`Include`、`Includes`、`LoadConfig`、`Modules`、`New`、`NewConfig`、`NewDecoder`、`NewEncoder`、`NewModules`、`ObjectFormat`、`OptBool`、`Option`、`Options`、`Paths`、`ReadConfig`、`RefSpec`、`RemoteConfig`、`RepositoryFormatVersion`、`Section`、`Sections`、`String`、`Submodule`、`Subsection`、`Subsections`

### `store`（39）

`AccountStoreBinder`、`ActiveAccountStore`、`ActiveGroupSnapshotter`、`CloudPrimaryStoreDeps`、`CloudSessionRepository`、`ConversationStore`、`EnsurePrivateDataDir`、`EventPage`、`EventPageQuery`、`EventReadMode`、`EventRecord`、`ExecutionReporter`、`GroupTombstoner`、`InMemoryConversationStore`、`Message`、`MessagePage`、`MessageQuery`、`MessageRole`、`NewCloudPrimaryStoreBinder`、`NewCloudSessionRepository`、`NewInMemoryConversationStore`、`NewRolloutChunkClient`、`NewRolloutConversationStore`、`NewRolloutConversationStoreFromParts`、`PrepareCloudPrimaryStores`、`RolloutCloudPrimaryStores`、`RolloutConversationStoreConfig`、`RunPreparation`、`Session`、`SessionEnvironment`、`SessionPage`、`SessionQuery`、`SessionReader`、`SessionRepository`、`ShouldPersistEventType`、`StageResult`、`StagedAccountStore`、`StartSessionRequest`、`SyncExecutionReporter`

### `server`（8）

`AssembleMemoryPipelineWithPaths`、`Loader`、`NewFilesystemLoader`、`NewPhaseSessionStarter`、`NewServer`、`NewWithConfigPathAndDataDir`、`WithClientType`、`WithHostOverridesSource`

### `protocol`（119）

`AccountLoginStartRequest`、`AccountLoginStartResult`、`AccountLogoutResult`、`AccountStatusResult`、`AccountUpdatedNotification`、`AgentInput`、`AttachSessionID`、`AuthMode`、`AuthRefreshRequest`、`AuthRefreshResponse`、`BashApproval`、`BashAuthRequest`、`BashAuthResponse`、`CanSetIdempotencyToken`、`ClientCapabilities`、`ClientInfo`、`ConfigReadRequest`、`ConfigReadResult`、`ConfigWriteRequest`、`ConfigWriteResult`、`ConnectorChanges`、`ConnectorCheckRequest`、`ConnectorCheckResult`、`ConnectorInfo`、`ConnectorManageRequest`、`ConnectorManageResponse`、`ConnectorResolveRequest`、`ConnectorResolveResponse`、`DecodeJSONValue`、`DocContext`、`DocSelection`、`EditedCanvasFile`、`EncodeJSONValue`、`ErrorEventData`、`ErrorUnmarshaler`、`EscapeMode`、`FormatTime`、`GetIdempotencyToken`、`HandlerPayloadUnmarshal`、`HostPrefixBuilder`、`IMMemoryCompactRequest`、`IMMemoryCompactResult`、`InitializeRequest`、`InitializeResult`、`JSONRPCMessage`、`MessageListRequest`、`MessageListResult`、`ModeTransition`、`ModelRef`、`NewConnectorManageRequest`、`NewErrorResponse`、`NewHostPrefixHandler`、`NewMessageErrorResponse`、`NewNotification`、`NewNotificationFromAGUIEvent`、`NewResponse`、`NewRunErrorNotification`、`NewRunScopedNotification`、`NewStreamBoundaryNotification`、`NewTimerTaskRequest`、`NewUnmarshalErrorHandler`、`ParseEditedCanvasFiles`、`ParseMessage`、`ParseTime`、`PayloadMarshaler`、`PayloadUnmarshaler`、`PermissionType`、`PlanReview`、`ResolvedConnector`、`ResponseMetadata`、`RunInterruptRequest`、`RunMetadata`、`RunStartResult`、`RunSteerResult`、`SandboxAccountInstallResult`、`SandboxAccountState`、`SandboxAccountStatusResult`、`SandboxAccountUninstallResult`、`ServerCapabilities`、`ServerInfo`、`SessionDeleteRequest`、`SessionDeleteResult`、`SessionEnvironment`、`SessionListOffsetResult`、`SessionListRequest`、`SessionListResult`、`SessionMemory`、`SessionMemoryItem`、`SessionPinRequest`、`SessionPinResult`、`SessionReadRequest`、`SessionReadResult`、`SessionSkill`、`SessionSkills`、`SessionSubscribeRequest`、`SessionSubscribeResult`、`SessionSubscriptionEndNotification`、`SessionSubscriptionEventNotification`、`SessionSummary`、`SessionUnsubscribeRequest`、`SessionUnsubscribeResult`、`SessionWorkspace`、`SkillListContextRequest`、`SkillListContextResponse`、`SteerAcceptedEvent`、`SteerConsumedEvent`、`SteerFailedEvent`、`StoredEvent`、`StreamStartNotification`、`TimerTaskRequest`、`TimerTaskResponse`、`UUIDVersion4`、`UnmarshalDiscardBody`、`UnmarshalErrorAssumeRoleWithWebIdentityInputcore`、`UnmarshalErrorHandler`、`ValidHostLabel`、`ValidPortNumber`、`ValidateEndpointHost`、`ValidateEndpointHostHandlerknownhosts`

### `rpc`（7）

`BadRequest`、`ErrorInfo`、`Health`、`Help`、`LocalizedMessageR`、`PreconditionFailure`、`QuotaFailure`

### `prompt`（416）

`AgentModeBuildReminder`、`AgentModeExitPlanReminder`、`AskUserQuestionHeaderArgDesc`、`AskUserQuestionInvalidOptionsCountResp`、`AskUserQuestionInvalidQuestionsCountResp`、`AskUserQuestionMissingHeaderResp`、`AskUserQuestionMissingQuestionResp`、`AskUserQuestionMultiSelectArgDesc`、`AskUserQuestionOptionDescriptionPropDesc`、`AskUserQuestionOptionLabelPropDesc`、`AskUserQuestionOptionMissingFieldResp`、`AskUserQuestionOptionsArgDesc`、`AskUserQuestionParseArgsFailedResp`、`AskUserQuestionQuestionArgDesc`、`AskUserQuestionQuestionsArgDesc`、`AskUserQuestionResp`、`AskUserQuestionToolDesc`、`AwaitBriefArgDesc`、`AwaitCancelledResp`、`AwaitCompletedResp`、`AwaitExecErrorResp`、`AwaitInvalidParamsResp`、`AwaitMaxTokensArgDesc`、`AwaitNotFoundResp`、`AwaitOutputLimitResp`、`AwaitTaskIDArgDesc`、`AwaitTerminateArgDesc`、`AwaitTerminatedResp`、`AwaitTimeoutArgDesc`、`AwaitTimeoutResp`、`AwaitToolDesc`、`BashBriefArgDesc`、`BashCommandArgDesc`、`BashEmptyCommandResp`、`BashErrorResp`、`BashIdleTimeoutResp`、`BashJustificationArgDesc`、`BashParseArgsFailedResp`、`BashParseFailedDeniedErr`、`BashParseFailedDeniedResp`、`BashRunInBackgroundArgDesc`、`BashSandboxPermissionsArgDesc`、`BashSuccessResp`、`BashToolDesc`、`BuildWikiDirTree`、`CompactBrief`、`CompactError`、`CompactPrompt`、`CompactWorkflowReminder`、`ConnectorExecActionArgDesc`、`ConnectorExecArgumentsArgDesc`、`ConnectorExecBriefArgDesc`、`ConnectorExecCallMissingToolResp`、`ConnectorExecIDArgDesc`、`ConnectorExecInvalidActionResp`、`ConnectorExecMissingActionResp`、`ConnectorExecMissingIDResp`、`ConnectorExecOperationFailedResp`、`ConnectorExecParseArgsFailedResp`、`ConnectorExecToolArgDesc`、`ConnectorExecToolDesc`、`ConnectorManageActionArgDesc`、`ConnectorManageActionMissingIDResp`、`ConnectorManageAddMissingModeResp`、`ConnectorManageAddMissingNameResp`、`ConnectorManageBriefArgDesc`、`ConnectorManageCommandArgDesc`、`ConnectorManageDescriptionArgDesc`、`ConnectorManageEndpointArgDesc`、`ConnectorManageHTTPMissingEndpointResp`、`ConnectorManageHeadersArgDesc`、`ConnectorManageIDArgDesc`、`ConnectorManageInvalidActionResp`、`ConnectorManageMissingActionResp`、`ConnectorManageModeArgDesc`、`ConnectorManageNameArgDesc`、`ConnectorManageOperationFailedResp`、`ConnectorManageParseArgsFailedResp`、`ConnectorManageStdioMissingCommandResp`、`ConnectorManageToolDesc`、`ConnectorManageUnsupportedModeResp`、`CreatePlanActionArgDesc`、`CreatePlanEditAllArgDesc`、`CreatePlanEditEmptyResp`、`CreatePlanEditFindArgDesc`、`CreatePlanEditNotFoundResp`、`CreatePlanEditReplaceArgDesc`、`CreatePlanEditsArgDesc`、`CreatePlanInvalidActionResp`、`CreatePlanInvalidTodosResp`、`CreatePlanMissingNameForEditResp`、`CreatePlanMissingNameResp`、`CreatePlanMissingOverviewResp`、`CreatePlanMissingPlanResp`、`CreatePlanNameArgDesc`、`CreatePlanOverviewArgDesc`、`CreatePlanParseArgsFailedResp`、`CreatePlanPlanArgDesc`、`CreatePlanReadFileFailedResp`、`CreatePlanResp`、`CreatePlanTodoContentArgDesc`、`CreatePlanTodoIDArgDesc`、`CreatePlanTodosArgDesc`、`CreatePlanToolDesc`、`CreatePlanWriteFileFailedResp`、`EditMemoryConflictResp`、`EditMemoryContentTooLargeResp`、`EditMemoryGetFailedResp`、`EditMemoryMultipleMatchesResp`、`EditMemoryNotFoundResp`、`EditMemoryOldNotFoundResp`、`EditMemorySetFailedResp`、`EditMemorySuccessResp`、`EditMemoryToolDesc`、`EnvironmentWorkspaceRoot`、`ExecAuthDeniedCommand`、`ExecAuthEscalationWithoutJustificationResp`、`ExecAuthInvalidSandboxPermissionsResp`、`ExecAuthJustificationWithoutEscalationResp`、`ExecAuthMismatchResp`、`ExecAuthPermissionDeniedErr`、`ExecAuthPermissionDeniedResp`、`ExecAuthRepeatResp`、`ExecAuthUserRejectedResp`、`ExecAuthUserSkippedResp`、`ExecErrMessage`、`ExecuteJupyterBriefArgDesc`、`ExecuteJupyterCodeArgDesc`、`ExecuteJupyterEmptyCodeUserResp`、`ExecuteJupyterErrorEmptyCodeResp`、`ExecuteJupyterErrorKernelResp`、`ExecuteJupyterKernelFailedUserResp`、`ExecuteJupyterParseArgsFailedResp`、`ExecuteJupyterParseArgsFailedUserResp`、`ExecuteJupyterSessionStateReminder`、`ExecuteJupyterSuccessNoOutputResp`、`ExecuteJupyterSuccessPersistedReminder`、`ExecuteJupyterSuccessReminder`、`ExecuteJupyterToolDesc`、`FileActionArgDesc`、`FileBriefArgDesc`、`FileContentArgDesc`、`FileContentRequiredResp`、`FileEditAllArgDesc`、`FileEditFailedResp`、`FileEditFindArgDesc`、`FileEditItemFindRequiredResp`、`FileEditNotFoundResp`、`FileEditPartialResp`、`FileEditReplaceArgDesc`、`FileEditSuccessResp`、`FileEditsArgDesc`、`FileEditsRequiredResp`、`FileEndLineArgDesc`、`FileInvalidLineRangeResp`、`FileMissingActionResp`、`FileOperationFailedUserResp`、`FileParseArgsFailedResp`、`FilePathArgDesc`、`FilePathRequiredResp`

### `dw`（22）

`AccumulatorFromAttributes`、`AccumulatorFromContext`、`BatchRequest`、`BatchResponse`、`BuildEnvelopes`、`EnsureRunContext`、`EventEnvelope`、`NewRunDWAccumulator`、`NewSpanCollector`、`Policy`、`ReportMode`、`ReportModeFromContext`、`Reporter`、`RequestSummary`、`RunDWAccumulator`、`RunOutcome`、`RunSummary`、`SpanCollector`、`SpanDTO`、`WithAccumulatorContext`、`WithRunContext`、`WithRunMetadata`

### `nativehost`（16）

`CDPError`、`CDPRequest`、`CDPResponse`、`EmbeddedServer`、`KillStaleNativeHost`、`ListenWithCleanup`、`NMInbound`、`NMOutbound`、`NewServer`、`PortCleanupResult`、`PortFilePath`、`Server`、`StartServer`、`StatusResponse`、`UnmarshalNM`、`WriteNM`

### `wpsclibroker`（2）

`Host`、`StartHost`

### `experts`（7）

`ExpertRunSnapshot`、`ExpertSkill`、`ExpertSpec`、`IsValidExpertID`、`LocalPackageLoader`、`NewPackageLoader`、`PackageLoader`
