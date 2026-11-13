# 灵犀 Electron IPC 通道全表（常量表抽取）

- 源: `C:\AI\wubigrok\re-lingxi\out\x\out\main\index.js`
- 形态: `t.AgentSteerRun="agent:steerRun"`（bundle 内导出的通道常量表）
- 命中 510 次 → 去重 **510** 个通道，**45** 个命名空间

## 命名空间 × 通道数

| 命名空间 | 通道数 |
|---|---|
| `file` | 57 |
| `agent` | 52 |
| `fileBrowser` | 41 |
| `iab` | 35 |
| `base` | 33 |
| `pet` | 27 |
| `window` | 23 |
| `business` | 22 |
| `project` | 21 |
| `auth` | 19 |
| `debugCenter` | 18 |
| `im` | 13 |
| `localScheduledTask` | 12 |
| `upload` | 12 |
| `remoteControl` | 10 |
| `logAnalysis` | 9 |
| `cowork` | 8 |
| `sandbox` | 8 |
| `notification` | 7 |
| `dev` | 6 |
| `settings` | 6 |
| `shortcuts` | 6 |
| `telemetryAcceptance` | 6 |
| `wikiCompile` | 6 |
| `pythonServer` | 5 |
| `fileAssociation` | 4 |
| `network` | 4 |
| `rendererTasks` | 4 |
| `resourcePreview` | 4 |
| `shell` | 4 |
| `update` | 4 |
| `dialog` | 3 |
| `loading` | 3 |
| `storage` | 3 |
| `appCheck` | 2 |
| `memoryExtraction` | 2 |
| `mic` | 2 |
| `skills` | 2 |
| `browser` | 1 |
| `deeplink` | 1 |
| `expert` | 1 |
| `feedback` | 1 |
| `hotUpdate` | 1 |
| `kdocs` | 1 |
| `nm` | 1 |

## 全量（常量名 → 通道）

### `agent`

- `AgentAcquirePreventSleep` → `agent:acquirePreventSleep`
- `AgentAnswerQuestion` → `agent:answerQuestion`
- `AgentAppServerStatusChanged` → `agent:appServerStatusChanged`
- `AgentCanvasEditEnded` → `agent:canvasEditEnded`
- `AgentCreateSession` → `agent:createSession`
- `AgentDeleteSession` → `agent:deleteSession`
- `AgentDisconnect` → `agent:disconnect`
- `AgentEvent` → `agent:event`
- `AgentFirstAiRequestReportingNeeded` → `agent:firstAiRequestReportingNeeded`
- `AgentForwardStreamEvent` → `agent:forwardStreamEvent`
- `AgentGetActiveRunId` → `agent:getActiveRunId`
- `AgentGetAppServerStatus` → `agent:getAppServerStatus`
- `AgentGetCompletionState` → `agent:getCompletionState`
- `AgentGetSessionList` → `agent:getSessionList`
- `AgentGetSessionMessages` → `agent:getSessionMessages`
- `AgentGetSessionQueueContext` → `agent:getSessionQueueContext`
- `AgentGetSessions` → `agent:getSessions`
- `AgentIMMemoryCompact` → `agent:immemoryCompact`
- `AgentPatchTrayRecentSessionPin` → `agent:patchTrayRecentSessionPin`
- `AgentPatchTrayRecentSessionTitle` → `agent:patchTrayRecentSessionTitle`
- `AgentQueueCapabilities` → `agent:queue:capabilities`
- `AgentQueueChanged` → `agent:queue:changed`
- `AgentQueueClaimOperation` → `agent:queue:claimOperation`
- `AgentQueueCommand` → `agent:queue:command`
- `AgentQueueExecutor` → `agent:queue:executor`
- `AgentQueuePreparationFailed` → `agent:queue:preparationFailed`
- `AgentQueueReportOperation` → `agent:queue:reportOperation`
- `AgentQueueReserveInput` → `agent:queue:reserveInput`
- `AgentQueueSendNow` → `agent:queue:sendNow`
- `AgentQueueSubscribe` → `agent:queue:subscribe`
- `AgentQueueUiFinished` → `agent:queue:uiFinished`
- `AgentQueueUnsubscribe` → `agent:queue:unsubscribe`
- `AgentReleasePreventSleep` → `agent:releasePreventSleep`
- `AgentRemoveTrayRecentSession` → `agent:removeTrayRecentSession`
- `AgentSendMessage` → `agent:sendMessage`
- `AgentSessionRunStarted` → `agent:sessionRunStarted`
- `AgentSessionRunStateChanged` → `agent:sessionRunStateChanged`
- `AgentSessionSubscribe` → `agent:sessionSubscribe`
- `AgentSessionSubscriptionEvent` → `agent:sessionSubscriptionEvent`
- `AgentSessionTitleUpdated` → `agent:sessionTitleUpdated`
- `AgentSessionUnsubscribe` → `agent:sessionUnsubscribe`
- `AgentSteerRun` → `agent:steerRun`
- `AgentStop` → `agent:stop`
- `AgentStopCompletion` → `agent:stopCompletion`
- `AgentTestModel` → `agent:testModel`
- `AgentWsClose` → `agent:ws:close`
- `AgentWsConnect` → `agent:ws:connect`
- `AgentWsError` → `agent:ws:error`
- `AgentWsMessage` → `agent:ws:message`
- `AgentWsMeta` → `agent:ws:meta`
- `AgentWsSend` → `agent:ws:send`
- `AgentWsUpgradeError` → `agent:ws:upgradeError`

### `appCheck`

- `AppCheckGetSnapshot` → `appCheck:getSnapshot`
- `AppCheckRunChecks` → `appCheck:runChecks`

### `auth`

- `AuthAccountListChanged` → `auth:accountListChanged`
- `AuthCheckInviteCode` → `auth:checkInviteCode`
- `AuthGetAccountList` → `auth:getAccountList`
- `AuthGetBootstrapState` → `auth:getBootstrapState`
- `AuthGetCachedSwitchUsers` → `auth:getCachedSwitchUsers`
- `AuthGetSwitchUsers` → `auth:getSwitchUsers`
- `AuthInviteCodeDialogClosed` → `auth:inviteCodeDialogClosed`
- `AuthLoginSuccess` → `auth:loginSuccess`
- `AuthLoginViaWeb` → `auth:loginViaWeb`
- `AuthLogout` → `auth:logout`
- `AuthNeedInviteCode` → `auth:needInviteCode`
- `AuthOpenLoginWindow` → `auth:openLoginWindow`
- `AuthRequestInviteCode` → `auth:requestInviteCode`
- `AuthRequestRealName` → `auth:requestRealName`
- `AuthRetryVerification` → `auth:retryVerification`
- `AuthSaveInviteCode` → `auth:saveInviteCode`
- `AuthStateChanged` → `auth:stateChanged`
- `AuthStateChangedV2` → `auth:stateChanged:v2`
- `AuthSwitchUser` → `auth:switchUser`

### `base`

- `BaseCancelUpdateDownload` → `base:cancelUpdateDownload`
- `BaseCheckUpdate` → `base:checkUpdate`
- `BaseClientUpdateDialogStateChanged` → `base:clientUpdateDialogStateChanged`
- `BaseCloseClientUpdateDialog` → `base:closeClientUpdateDialog`
- `BaseExportLogs` → `base:exportLogs`
- `BaseForceUpdateRequired` → `base:forceUpdateRequired`
- `BaseForceUpdateTaskCheckRequested` → `base:forceUpdateTaskCheckRequested`
- `BaseForceUpdateTaskCheckResponded` → `base:forceUpdateTaskCheckResponded`
- `BaseGetAppDataPath` → `base:getAppDataPath`
- `BaseGetAppInfo` → `base:getAppInfo`
- `BaseGetAppMemoryUsage` → `base:getAppMemoryUsage`
- `BaseGetClientUpdateDialogInfo` → `base:getClientUpdateDialogInfo`
- `BaseGetClientUpdateRuntimeState` → `base:getClientUpdateRuntimeState`
- `BaseGetDeviceInfo` → `base:getDeviceInfo`
- `BaseGetVersion` → `base:getVersion`
- `BaseLog` → `base:log`
- `BaseReportRendererReady` → `base:reportRendererReady`
- `BaseReportUpdateSidebarClick` → `base:reportUpdateSidebarClick`
- `BaseReportWebVersion` → `base:reportWebVersion`
- `BaseRestartAndInstallUpdate` → `base:restartAndInstallUpdate`
- `BaseRestartApp` → `base:restartApp`
- `BaseShowClientUpdateDialog` → `base:showClientUpdateDialog`
- `BaseStartPreparedClientUpdate` → `base:startPreparedClientUpdate`
- `BaseStartUpdate` → `base:startUpdate`
- `BaseSyncMainContentWebEnv` → `base:syncMainContentWebEnv`
- `BaseTaskInterruptionDialogCancelled` → `base:taskInterruptionDialogCancelled`
- `BaseTaskInterruptionDialogRequested` → `base:taskInterruptionDialogRequested`
- `BaseTaskInterruptionDialogResponded` → `base:taskInterruptionDialogResponded`
- `BaseTaskInterruptionReportRequested` → `base:taskInterruptionReportRequested`
- `BaseUpdateResourceDialogCancelled` → `base:updateResourceDialogCancelled`
- `BaseUpdateResourceDialogRequested` → `base:updateResourceDialogRequested`
- `BaseUpdateResourceDialogResponded` → `base:updateResourceDialogResponded`
- `BaseUpdateStatusChanged` → `base:updateStatusChanged`

### `browser`

- `BrowserCreatePage` → `browser:createPage`

### `business`

- `BusinessConnectorsChanged` → `business:connectorsChanged`
- `BusinessGetConnector` → `business:getConnector`
- `BusinessGetConnectors` → `business:getConnectors`
- `BusinessGetGlobalConnectorState` → `business:getGlobalConnectorState`
- `BusinessGetSessionConnectorState` → `business:getSessionConnectorState`
- `BusinessGetSkills` → `business:getSkills`
- `BusinessGetSkillsMeta` → `business:getSkillsMeta`
- `BusinessInstallSkillFiles` → `business:installSkillFiles`
- `BusinessInstallSkillFromUrl` → `business:installSkillFromUrl`
- `BusinessPackFolderToSkillZip` → `business:packFolderToSkillZip`
- `BusinessPackSkillZip` → `business:packSkillZip`
- `BusinessRemoveConnector` → `business:removeConnector`
- `BusinessRemoveSkill` → `business:removeSkill`
- `BusinessSetConnector` → `business:setConnector`
- `BusinessSetGlobalConnectorState` → `business:setGlobalConnectorState`
- `BusinessSetSessionConnectorState` → `business:setSessionConnectorState`
- `BusinessSkillsChanged` → `business:skillsChanged`
- `BusinessTestConnector` → `business:testConnector`
- `BusinessUploadSkillZip` → `business:uploadSkillZip`
- `WorkspaceGetPath` → `business:workspaceGetPath`
- `WorkspaceSetPath` → `business:workspaceSetPath`
- `WorkspaceValidatePath` → `business:workspaceValidatePath`

### `cowork`

- `CoworkGetAccessToken` → `cowork:getAccessToken`
- `CoworkSandboxUploadProgress` → `cowork:sandboxUploadProgress`
- `CoworkSandboxUploadRegister` → `cowork:sandboxUploadRegister`
- `CoworkSandboxUploadUnregister` → `cowork:sandboxUploadUnregister`
- `CoworkSessionShareCancel` → `cowork:sessionShareCancel`
- `CoworkSessionShareProgress` → `cowork:sessionShareProgress`
- `CoworkSessionShareQuery` → `cowork:sessionShareQuery`
- `CoworkSessionShareStart` → `cowork:sessionShareStart`

### `debugCenter`

- `DebugCenterApplyRuntimeConfig` → `debugCenter:applyRuntimeConfig`
- `DebugCenterGetMemoryDebugVisible` → `debugCenter:getMemoryDebugVisible`
- `DebugCenterGetRendererManifest` → `debugCenter:getRendererManifest`
- `DebugCenterGetRendererRegistry` → `debugCenter:getRendererRegistry`
- `DebugCenterGetRuntimeConfig` → `debugCenter:getRuntimeConfig`
- `DebugCenterMemoryDebugVisibilityChanged` → `debugCenter:memoryDebugVisibilityChanged`
- `DebugCenterOpenExternalFiles` → `debugCenter:openExternalFiles`
- `DebugCenterReplaceAppServerArchive` → `debugCenter:replaceAppServerArchive`
- `DebugCenterResetAppServer` → `debugCenter:resetAppServer`
- `DebugCenterRestartToApply` → `debugCenter:restartToApply`
- `DebugCenterSetMemoryDebugVisible` → `debugCenter:setMemoryDebugVisible`
- `DebugCenterSetRendererRegistryOverride` → `debugCenter:setRendererRegistryOverride`
- `DebugCenterSetServerBranch` → `debugCenter:setServerBranch`
- `DebugCenterSetTelemetryAcceptance` → `debugCenter:setTelemetryAcceptance`
- `DebugCenterSetTrackingConsole` → `debugCenter:setTrackingConsole`
- `DebugCenterTestProxy` → `debugCenter:testProxy`
- `DebugCenterToggle` → `debugCenter:toggle`
- `DebugCenterVerifyRendererAsarAndRestart` → `debugCenter:verifyRendererAsarAndRestart`

### `deeplink`

- `DeeplinkNavigate` → `deeplink:navigate`

### `dev`

- `DevChannelAttributionLog` → `dev:channelAttributionLog`
- `DevRequestLog` → `dev:requestLog`
- `DevRequestMockClearAll` → `dev:requestMock:clearAll`
- `DevRequestMockDeleteRule` → `dev:requestMock:deleteRule`
- `DevRequestMockGetRules` → `dev:requestMock:getRules`
- `DevRequestMockSaveRule` → `dev:requestMock:saveRule`

### `dialog`

- `DialogGetPendingPayload` → `dialog:getPendingPayload`
- `DialogResizeToContent` → `dialog:resizeToContent`
- `DialogResolve` → `dialog:resolve`

### `expert`

- `ExpertPrepare` → `expert:prepare`

### `feedback`

- `FeedbackSubmitLingxi` → `feedback:submitLingxi`

### `file`

- `FsCopyFile` → `file:copyFile`
- `FsCreateDirectory` → `file:createDirectory`
- `FsDeletePath` → `file:deletePath`
- `FsDirChanged` → `file:dirChanged`
- `FsEditFile` → `file:editFile`
- `FsEditedCanvasAck` → `file:editedCanvasAck`
- `FsEditedCanvasList` → `file:editedCanvasList`
- `FsEditedCanvasMarkPreviewed` → `file:editedCanvasMarkPreviewed`
- `FsEditedCanvasMarkRevision` → `file:editedCanvasMarkRevision`
- `FsEditedCanvasRemoveSession` → `file:editedCanvasRemoveSession`
- `FsEditedCanvasSetBaseline` → `file:editedCanvasSetBaseline`
- `FsGetImagePreviewPolicy` → `file:getImagePreviewPolicy`
- `FsGrantLocalPreviewPath` → `file:grantLocalPreviewPath`
- `FsHtmlThumbnailEnsure` → `file:htmlThumbnailEnsure`
- `FsHtmlThumbnailRead` → `file:htmlThumbnailRead`
- `FsHtmlThumbnailReady` → `file:htmlThumbnailReady`
- `FsListAppsForFile` → `file:listAppsForFile`
- `FsListDir` → `file:listDir`
- `FsLocalPreviewStateChanged` → `file:localPreviewStateChanged`
- `FsOpenDirectory` → `file:openDirectory`
- `FsOpenFile` → `file:openFile`
- `FsOpenFileWithApp` → `file:openFileWithApp`
- `FsReadEditableText` → `file:readEditableText`
- `FsReadFile` → `file:readFile`
- `FsReadFileAsBuffer` → `file:readFileAsBuffer`
- `FsReadImageThumbnail` → `file:readImageThumbnail`
- `FsReadPreview` → `file:readPreview`
- `FsRecheckLocalPreviewScope` → `file:recheckLocalPreviewScope`
- `FsReleaseLocalPreviewScope` → `file:releaseLocalPreviewScope`
- `FsRenamePath` → `file:renamePath`
- `FsReplaceLocalPreviewScope` → `file:replaceLocalPreviewScope`
- `FsSaveCloudDriveFile` → `file:saveCloudDriveFile`
- `FsSaveEditableText` → `file:saveEditableText`
- `FsSaveEditableTextAs` → `file:saveEditableTextAs`
- `FsSaveOnlineFile` → `file:saveOnlineFile`
- `FsSavePasteFile` → `file:savePasteFile`
- `FsSearchCompleted` → `file:searchCompleted`
- `FsSearchStart` → `file:searchStart`
- `FsSearchStop` → `file:searchStop`
- `FsSearchUpdate` → `file:searchUpdate`
- `FsSearchUpdated` → `file:searchUpdated`
- `FileSelectFile` → `file:selectFile`
- `FileSelectMultiFolder` → `file:selectMultiFolder`
- `FileSelectSingleFolder` → `file:selectSingleFolder`
- `FsShowItemInFolder` → `file:showItemInFolder`
- `FileShowSaveDialog` → `file:showSaveDialog`
- `FsSyncBindingRemove` → `file:syncBindingRemove`
- `FsSyncBindingUpsert` → `file:syncBindingUpsert`
- `FsSyncBindingsList` → `file:syncBindingsList`
- `FsSyncCreateConflictCopy` → `file:syncCreateConflictCopy`
- `FsSyncFingerprintLocal` → `file:syncFingerprintLocal`
- `FsSyncFingerprintOnline` → `file:syncFingerprintOnline`
- `FsSyncPullOnline` → `file:syncPullOnline`
- `FsUnwatchDir` → `file:unwatchDir`
- `FsUpdateImagePreviewContext` → `file:updateImagePreviewContext`
- `FsWatchDir` → `file:watchDir`
- `FsWriteFile` → `file:writeFile`

### `fileAssociation`

- `FileAssociationDefaultAppsSettingsReturnedForWindows` → `fileAssociation:defaultAppsSettingsReturnedForWindows`
- `FileAssociationGetDefaultHandlers` → `fileAssociation:getDefaultHandlers`
- `FileAssociationOpenDefaultAppsSettingsForWindows` → `fileAssociation:openDefaultAppsSettingsForWindows`
- `FileAssociationSetDefaultHandlersForMacOS` → `fileAssociation:setDefaultHandlersForMacOS`

### `fileBrowser`

- `FileBrowserActivateTab` → `fileBrowser:activateTab`
- `FileBrowserCloseOtherTabs` → `fileBrowser:closeOtherTabs`
- `FileBrowserCloseTab` → `fileBrowser:closeTab`
- `FileBrowserContentFailed` → `fileBrowser:contentFailed`
- `FileBrowserContentReady` → `fileBrowser:contentReady`
- `FileBrowserCreateIabTab` → `fileBrowser:createIabTab`
- `FileBrowserGetState` → `fileBrowser:getState`
- `FileBrowserGetTab` → `fileBrowser:getTab`
- `FileBrowserHtmlFrame` → `fileBrowser:htmlFrame`
- `FileBrowserIabCapturePage` → `fileBrowser:iabCapturePage`
- `FileBrowserIabClearCache` → `fileBrowser:iabClearCache`
- `FileBrowserIabClearCookies` → `fileBrowser:iabClearCookies`
- `FileBrowserIabGetZoom` → `fileBrowser:iabGetZoom`
- `FileBrowserIabGuestPointerDown` → `fileBrowser:iabGuestPointerDown`
- `FileBrowserIabImportChromeCookies` → `fileBrowser:iabImportChromeCookies`
- `FileBrowserIabSetZoom` → `fileBrowser:iabSetZoom`
- `FileBrowserIsSupportedFilePath` → `fileBrowser:isSupportedFilePath`
- `FileBrowserMainSettingsRequested` → `fileBrowser:mainSettingsRequested`
- `FileBrowserModalStateChanged` → `fileBrowser:modalStateChanged`
- `FileBrowserMoveTab` → `fileBrowser:moveTab`
- `FileBrowserOpenDevTools` → `fileBrowser:openDevTools`
- `FileBrowserOpenFiles` → `fileBrowser:openFiles`
- `FileBrowserOpenMainWindow` → `fileBrowser:openMainWindow`
- `FileBrowserOpenUrl` → `fileBrowser:openUrl`
- `FileBrowserOpenWithSystem` → `fileBrowser:openWithSystem`
- `FileBrowserReloadTab` → `fileBrowser:reloadTab`
- `FileBrowserRendererReady` → `fileBrowser:rendererReady`
- `FileBrowserReorderTab` → `fileBrowser:reorderTab`
- `FileBrowserResolveIabNavigation` → `fileBrowser:resolveIabNavigation`
- `FileBrowserResolveSave` → `fileBrowser:resolveSave`
- `FileBrowserSaveAs` → `fileBrowser:saveAs`
- `FileBrowserSaveRequested` → `fileBrowser:saveRequested`
- `FileBrowserSessionActivityChanged` → `fileBrowser:sessionActivityChanged`
- `FileBrowserSetSessionId` → `fileBrowser:setSessionId`
- `FileBrowserShowItemInFolder` → `fileBrowser:showItemInFolder`
- `FileBrowserStateChanged` → `fileBrowser:stateChanged`
- `FileBrowserTabActiveChanged` → `fileBrowser:tabActiveChanged`
- `FileBrowserThemeChanged` → `fileBrowser:themeChanged`
- `FileBrowserTogglePin` → `fileBrowser:togglePin`
- `FileBrowserUpdateIabRuntime` → `fileBrowser:updateIabRuntime`
- `FileBrowserUpdateTabRuntime` → `fileBrowser:updateTabRuntime`

### `hotUpdate`

- `HotUpdateManualCheck` → `hotUpdate:manualCheck`

### `iab`

- `IabCancelDownload` → `iab:cancelDownload`
- `IabCaptureVisibleTab` → `iab:captureVisibleTab`
- `IabClearCache` → `iab:clearCache`
- `IabClearCookies` → `iab:clearCookies`
- `IabClearDownloadHistory` → `iab:clearDownloadHistory`
- `IabCloseTab` → `iab:closeTab`
- `IabDismissDownload` → `iab:dismissDownload`
- `IabFindInPage` → `iab:findInPage`
- `IabGetState` → `iab:getState`
- `IabGetZoom` → `iab:getZoom`
- `IabHide` → `iab:hide`
- `IabImportChromeCookies` → `iab:importChromeCookies`
- `IabListChromeProfiles` → `iab:listChromeProfiles`
- `IabListDownloads` → `iab:listDownloads`
- `IabNavigate` → `iab:navigate`
- `IabOpen` → `iab:open`
- `IabOpenDownload` → `iab:openDownload`
- `IabOpenDownloadsFolder` → `iab:openDownloadsFolder`
- `IabPauseDownload` → `iab:pauseDownload`
- `IabProbeComplete` → `iab:probe:complete`
- `IabProbeDestroyGuest` → `iab:probe:destroyGuest`
- `IabProbeMountGuest` → `iab:probe:mountGuest`
- `IabProbeProgress` → `iab:probe:progress`
- `IabProbeRegisterGuest` → `iab:probe:registerGuest`
- `IabProbeStart` → `iab:probe:start`
- `IabRegisterGuest` → `iab:registerGuest`
- `IabResumeDownload` → `iab:resumeDownload`
- `IabRevealDownload` → `iab:revealDownload`
- `IabSelectTab` → `iab:selectTab`
- `IabSetVisible` → `iab:setVisible`
- `IabSetZoom` → `iab:setZoom`
- `IabStopFindInPage` → `iab:stopFindInPage`
- `IabStopSession` → `iab:stopSession`
- `IabUnregisterGuest` → `iab:unregisterGuest`
- `IabUnregisterGuestSync` → `iab:unregisterGuestSync`

### `im`

- `ImClearAllSessionData` → `im:clearAllSessionData`
- `ImClearImSessionMap` → `im:clearImSessionMap`
- `ImConnectionStatusChanged` → `im:connectionStatusChanged`
- `ImGetChannelList` → `im:getChannelList`
- `ImGetMessages` → `im:getMessages`
- `ImGetSessionInfo` → `im:getSessionInfo`
- `ImMessagesChanged` → `im:messagesChanged`
- `ImRunEnded` → `im:runEnded`
- `ImSetUnifiedSessionId` → `im:setUnifiedSessionId`
- `ImWeixinQrCancel` → `im:weixinQrCancel`
- `ImWeixinQrLogout` → `im:weixinQrLogout`
- `ImWeixinQrStart` → `im:weixinQrStart`
- `ImWeixinQrStatus` → `im:weixinQrStatus`

### `kdocs`

- `KdocsGetUserInfo` → `kdocs:getUserInfo`

### `loading`

- `LoadingHide` → `loading:hide`
- `LoadingShow` → `loading:show`
- `LoadingShowError` → `loading:showError`

### `localScheduledTask`

- `LocalScheduledTaskChanged` → `localScheduledTask:changed`
- `LocalScheduledTaskDelete` → `localScheduledTask:delete`
- `LocalScheduledTaskExpired` → `localScheduledTask:expired`
- `LocalScheduledTaskGet` → `localScheduledTask:get`
- `LocalScheduledTaskHasActiveRuns` → `localScheduledTask:hasActiveRuns`
- `LocalScheduledTaskList` → `localScheduledTask:list`
- `LocalScheduledTaskListRuns` → `localScheduledTask:listRuns`
- `LocalScheduledTaskMarkRunsRead` → `localScheduledTask:markRunsRead`
- `LocalScheduledTaskRunFinished` → `localScheduledTask:runFinished`
- `LocalScheduledTaskRunNow` → `localScheduledTask:runNow`
- `LocalScheduledTaskSetEnabled` → `localScheduledTask:setEnabled`
- `LocalScheduledTaskUpsert` → `localScheduledTask:upsert`

### `logAnalysis`

- `LogAnalysisCancel` → `logAnalysis:cancel`
- `LogAnalysisEvent` → `logAnalysis:event`
- `LogAnalysisFollowUp` → `logAnalysis:followUp`
- `LogAnalysisGetCurrentRunLogInfo` → `logAnalysis:getCurrentRunLogInfo`
- `LogAnalysisGetLogsPath` → `logAnalysis:getLogsPath`
- `LogAnalysisOpenLogsDir` → `logAnalysis:openLogsDir`
- `LogAnalysisPickLogsDir` → `logAnalysis:pickLogsDir`
- `LogAnalysisResetLogsDir` → `logAnalysis:resetLogsDir`
- `LogAnalysisStart` → `logAnalysis:start`

### `memoryExtraction`

- `MemoryExtractionConfigRead` → `memoryExtraction:configRead`
- `MemoryExtractionConfigWrite` → `memoryExtraction:configWrite`

### `mic`

- `MicGetPermissionStatus` → `mic:getPermissionStatus`
- `MicRequestPermission` → `mic:requestPermission`

### `network`

- `NetworkGetDiagnostics` → `network:getDiagnostics`
- `NetworkGetReachability` → `network:getReachability`
- `NetworkReachabilityChanged` → `network:reachabilityChanged`
- `NetworkRunDiagnostics` → `network:runDiagnostics`

### `nm`

- `NmGetStatus` → `nm:getStatus`

### `notification`

- `NotificationDispatchTaskEvent` → `notification:dispatchTaskEvent`
- `NotificationOpenSystemSettings` → `notification:openSystemSettings`
- `NotificationRequestSystemPermission` → `notification:requestSystemPermission`
- `NotificationShow` → `notification:show`
- `NotificationTaskActivated` → `notification:taskActivated`
- `NotificationTaskActivationReady` → `notification:taskActivationReady`
- `NotificationTaskShowInApp` → `notification:taskShowInApp`

### `pet`

- `PetApplyCustomization` → `pet:applyCustomization`
- `PetAutoUpdateChanged` → `pet:autoUpdateChanged`
- `PetBundleProgressChanged` → `pet:bundleProgressChanged`
- `PetCheckAutoUpdate` → `pet:checkAutoUpdate`
- `PetCoworkAsk` → `pet:coworkAsk`
- `PetCoworkStop` → `pet:coworkStop`
- `PetDelete` → `pet:delete`
- `PetEnsureMaterial` → `pet:ensureMaterial`
- `PetGetAutoUpdateState` → `pet:getAutoUpdateState`
- `PetGetBinaryVersion` → `pet:getBinaryVersion`
- `PetGetIconPreview` → `pet:getIconPreview`
- `PetGetList` → `pet:getList`
- `PetGetPreferences` → `pet:getPreferences`
- `PetGetStatus` → `pet:getStatus`
- `PetLaunch` → `pet:launch`
- `PetNavigate` → `pet:navigate`
- `PetNotify` → `pet:notify`
- `PetOpenCustomDir` → `pet:openCustomDir`
- `PetOpenPetsDir` → `pet:openPetsDir`
- `PetRefresh` → `pet:refresh`
- `PetRename` → `pet:rename`
- `PetSelect` → `pet:select`
- `PetSendCommand` → `pet:sendCommand`
- `PetSetTheme` → `pet:setTheme`
- `PetStatusChanged` → `pet:statusChanged`
- `PetStop` → `pet:stop`
- `PetSwitchSkin` → `pet:switchSkin`

### `project`

- `ProjectAddFile` → `project:addFile`
- `ProjectAddWorkspace` → `project:addWorkspace`
- `ProjectClean` → `project:clean`
- `ProjectFilterWorkspacePaths` → `project:filterWorkspacePaths`
- `ProjectGetArtifactsPanelState` → `project:getArtifactsPanelState`
- `ProjectGetInputDraft` → `project:getInputDraft`
- `ProjectGetLastOpened` → `project:getLastOpened`
- `ProjectGetWorkspacePath` → `project:getWorkspacePath`
- `ProjectGetWorkspaces` → `project:getWorkspaces`
- `ProjectInit` → `project:init`
- `ProjectListFiles` → `project:listFiles`
- `ProjectRemoveFile` → `project:removeFile`
- `ProjectRemoveInputDraft` → `project:removeInputDraft`
- `ProjectRemoveWorkspace` → `project:removeWorkspace`
- `ProjectResolveSessionWorkspaces` → `project:resolveSessionWorkspaces`
- `ProjectSaveArtifactsPanelState` → `project:saveArtifactsPanelState`
- `ProjectSaveInputDraft` → `project:saveInputDraft`
- `ProjectSetLastOpened` → `project:setLastOpened`
- `ProjectSetWorkspaceCwd` → `project:setWorkspaceCwd`
- `ProjectSetWorkspaces` → `project:setWorkspaces`
- `ProjectSummary` → `project:summary`

### `pythonServer`

- `PythonServerGetStatus` → `pythonServer:getStatus`
- `PythonServerGetWsUrl` → `pythonServer:getWsUrl`
- `PythonServerStart` → `pythonServer:start`
- `PythonServerStatusChanged` → `pythonServer:statusChanged`
- `PythonServerStop` → `pythonServer:stop`

### `remoteControl`

- `RemoteControlBridgeStateChanged` → `remoteControl:bridgeStateChanged`
- `RemoteControlDeleteDevice` → `remoteControl:deleteDevice`
- `RemoteControlGetState` → `remoteControl:getState`
- `RemoteControlListDevices` → `remoteControl:listDevices`
- `RemoteControlReconnect` → `remoteControl:reconnect`
- `RemoteControlSessionAgentFrame` → `remoteControl:sessionAgentFrame`
- `RemoteControlSessionRunStarted` → `remoteControl:sessionRunStarted`
- `RemoteControlSessionStatus` → `remoteControl:sessionStatus`
- `RemoteControlSetForegroundSession` → `remoteControl:setForegroundSession`
- `RemoteControlShouldUseStreamCache` → `remoteControl:shouldUseStreamCache`

### `rendererTasks`

- `RendererTasksFinish` → `rendererTasks:finish`
- `RendererTasksRegister` → `rendererTasks:register`
- `RendererTasksTriggered` → `rendererTasks:triggered`
- `RendererTasksUnregister` → `rendererTasks:unregister`

### `resourcePreview`

- `ResourcePreviewClose` → `resourcePreview:close`
- `ResourcePreviewOpen` → `resourcePreview:open`
- `ResourcePreviewStateChanged` → `resourcePreview:stateChanged`
- `ResourcePreviewUpdate` → `resourcePreview:update`

### `sandbox`

- `SandboxDismissOnboarding` → `sandbox:dismissOnboarding`
- `SandboxGetStatus` → `sandbox:getStatus`
- `SandboxInstall` → `sandbox:install`
- `SandboxOnboardingRendered` → `sandbox:onboardingRendered`
- `SandboxReadConfig` → `sandbox:readConfig`
- `SandboxRetryRuntimeSetup` → `sandbox:retryRuntimeSetup`
- `SandboxStatusChanged` → `sandbox:statusChanged`
- `SandboxWriteConfig` → `sandbox:writeConfig`

### `settings`

- `SettingsGet` → `settings:get`
- `SettingsOpen` → `settings:open`
- `SettingsOpenDevLogWindow` → `settings:openDevLogWindow`
- `SettingsOpenLogAnalysisWindow` → `settings:openLogAnalysisWindow`
- `SettingsSet` → `settings:set`
- `SettingsSetSystemProxyEnabled` → `settings:setSystemProxyEnabled`

### `shell`

- `ShellNavigate` → `shell:navigate`
- `ShellOpenExternal` → `shell:openExternal`
- `ShellOpenPath` → `shell:openPath`
- `ShellOpenUrlInBrowser` → `shell:openUrlInBrowser`

### `shortcuts`

- `ShortcutsPetRegistrationChanged` → `shortcuts:petRegistrationChanged`
- `ShortcutsRegister` → `shortcuts:register`
- `ShortcutsSetRecordingMode` → `shortcuts:setRecordingMode`
- `ShortcutsTrigger` → `shortcuts:trigger`
- `ShortcutsTryRegister` → `shortcuts:tryRegister`
- `ShortcutsUnregister` → `shortcuts:unregister`

### `skills`

- `SkillsGetContent` → `skills:getContent`
- `SkillsSetEnabled` → `skills:setEnabled`

### `storage`

- `StorageClear` → `storage:clear`
- `StorageGetCleanupPreview` → `storage:getCleanupPreview`
- `StorageGetUsageSnapshot` → `storage:getUsageSnapshot`

### `telemetryAcceptance`

- `TelemetryAcceptanceClear` → `telemetryAcceptance:clear`
- `TelemetryAcceptanceEntry` → `telemetryAcceptance:entry`
- `TelemetryAcceptanceGetHistory` → `telemetryAcceptance:getHistory`
- `TelemetryAcceptanceRecord` → `telemetryAcceptance:record`
- `TelemetryAcceptanceRecordingEnabledChanged` → `telemetryAcceptance:recordingEnabledChanged`
- `TelemetryAcceptanceRendererReady` → `telemetryAcceptance:rendererReady`

### `update`

- `UnifiedUpdateAckRendererApplyFailure` → `update:ackRendererApplyFailure`
- `UnifiedUpdateGetState` → `update:getUnifiedState`
- `UnifiedUpdateRequestCheck` → `update:requestCheck`
- `UnifiedUpdateStateChanged` → `update:unifiedStateChanged`

### `upload`

- `UploadCancelBatch` → `upload:cancelBatch`
- `UploadCancelTask` → `upload:cancelTask`
- `UploadChanged` → `upload:changed`
- `UploadClearBatch` → `upload:clearBatch`
- `UploadCreateBatch` → `upload:createBatch`
- `UploadGetBatch` → `upload:getBatch`
- `UploadGrantFiles` → `upload:grantFiles`
- `UploadListBatches` → `upload:listBatches`
- `UploadPauseBatch` → `upload:pauseBatch`
- `UploadRemoveTask` → `upload:removeTask`
- `UploadResumeBatch` → `upload:resumeBatch`
- `UploadRetryTask` → `upload:retryTask`

### `wikiCompile`

- `WikiCompileGetConfig` → `wikiCompile:getConfig`
- `WikiCompileGetRuns` → `wikiCompile:getRuns`
- `WikiCompileGetStatus` → `wikiCompile:getStatus`
- `WikiCompileSetConfig` → `wikiCompile:setConfig`
- `WikiCompileStatusChanged` → `wikiCompile:statusChanged`
- `WikiCompileTrigger` → `wikiCompile:trigger`

### `window`

- `WindowCashierClosed` → `window:cashierClosed`
- `WindowClose` → `window:close`
- `WindowCloseCurrent` → `window:closeCurrent`
- `WindowCloseDevLogWindow` → `window:closeDevLogWindow`
- `WindowCloseLogAnalysisWindow` → `window:closeLogAnalysisWindow`
- `WindowCloseSkillsManager` → `window:closeSkillsManager`
- `WindowForegroundStateChanged` → `window:foregroundStateChanged`
- `WindowGetForegroundState` → `window:getForegroundState`
- `WindowGetIsMaximized` → `window:getIsMaximized`
- `WindowGetTitleBarInfo` → `window:getTitleBarInfo`
- `WindowMaximizeChanged` → `window:maximizeChanged`
- `WindowMinimize` → `window:minimize`
- `WindowOpenDevTools` → `window:openDevTools`
- `WindowOpenLogDirectory` → `window:openLogDirectory`
- `WindowOpenMainContentDevTools` → `window:openMainContentDevTools`
- `WindowOpenSkillsManager` → `window:openSkillsManager`
- `WindowReloadMainContent` → `window:reloadMainContent`
- `WindowSetDragRegionInset` → `window:setDragRegionInset`
- `WindowSetMainContentVisible` → `window:setMainContentVisible`
- `WindowSetTransparent` → `window:setTransparent`
- `WindowToggleMaximize` → `window:toggleMaximize`
- `WindowTransparentChanged` → `window:transparentChanged`
- `WindowVisibilityChanged` → `window:visibilityChanged`
