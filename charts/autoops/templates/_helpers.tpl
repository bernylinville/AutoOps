{{- define "autoops.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "autoops.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- include "autoops.name" . -}}
{{- end -}}
{{- end -}}

{{- define "autoops.labels" -}}
app.kubernetes.io/name: {{ include "autoops.name" . }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "autoops.selectorLabels" -}}
app.kubernetes.io/name: {{ include "autoops.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "autoops.apiServiceName" -}}
{{ include "autoops.fullname" . }}-api
{{- end -}}

{{- define "autoops.webServiceName" -}}
{{ include "autoops.fullname" . }}-web
{{- end -}}

{{- define "autoops.postgresServiceName" -}}
{{ include "autoops.fullname" . }}-postgres
{{- end -}}

{{- define "autoops.valkeyServiceName" -}}
{{ include "autoops.fullname" . }}-valkey
{{- end -}}

{{- define "autoops.secretName" -}}
{{- if eq (default "generated" .Values.secret.mode) "existing" -}}
{{- required "secret.existingSecret.name is required when secret.mode=existing" .Values.secret.existingSecret.name -}}
{{- else -}}
{{ include "autoops.fullname" . }}-secret
{{- end -}}
{{- end -}}

{{- define "autoops.secretKey" -}}
{{- $root := .root -}}
{{- $keys := $root.Values.secret.existingSecret.keys -}}
{{- $map := dict
  "dbPassword" (default "DB_PASSWORD" $keys.dbPassword)
  "redisPassword" (default "REDIS_PASSWORD" $keys.redisPassword)
  "jwtSecret" (default "JWT_SECRET" $keys.jwtSecret)
  "agentBearerToken" (default "AGENT_BEARER_TOKEN" $keys.agentBearerToken)
  "heartbeatToken" (default "HEARTBEAT_TOKEN" $keys.heartbeatToken)
  "flashdutyAppKey" (default "FLASHDUTY_APP_KEY" $keys.flashdutyAppKey)
  "flashdutyIntegrationKey" (default "FLASHDUTY_INTEGRATION_KEY" $keys.flashdutyIntegrationKey)
  "dingtalkWebhookURL" (default "DINGTALK_WEBHOOK_URL" $keys.dingtalkWebhookURL)
  "dingtalkApprovalClientSecret" (default "DINGTALK_APPROVAL_CLIENT_SECRET" $keys.dingtalkApprovalClientSecret)
  "deployBotWebhookURL" (default "DEPLOY_BOT_WEBHOOK_URL" $keys.deployBotWebhookURL)
  "deployBotSecret" (default "DEPLOY_BOT_SECRET" $keys.deployBotSecret)
-}}
{{- required (printf "unknown secret key mapping requested: %s" .key) (get $map .key) -}}
{{- end -}}

{{- define "autoops.storageClassField" -}}
{{- $persistence := . -}}
{{- if and $persistence.useDefaultStorageClass $persistence.storageClassName -}}
{{- fail "persistence.useDefaultStorageClass and persistence.storageClassName cannot both be set" -}}
{{- end -}}
{{- with $persistence.storageClassName }}
storageClassName: {{ . | quote }}
{{- end -}}
{{- end -}}

{{- define "autoops.uploadClaimName" -}}
{{- if .Values.upload.persistence.existingClaim -}}
{{ .Values.upload.persistence.existingClaim }}
{{- else -}}
{{ include "autoops.fullname" . }}-upload
{{- end -}}
{{- end -}}

{{- define "autoops.gitopsClaimName" -}}
{{- if .Values.gitopsWorkingTree.persistence.existingClaim -}}
{{ .Values.gitopsWorkingTree.persistence.existingClaim }}
{{- else -}}
{{ include "autoops.fullname" . }}-gitops
{{- end -}}
{{- end -}}

{{- define "autoops.postgresClaimName" -}}
{{- if .Values.postgres.persistence.existingClaim -}}
{{ .Values.postgres.persistence.existingClaim }}
{{- else -}}
{{ include "autoops.fullname" . }}-postgres
{{- end -}}
{{- end -}}

{{- define "autoops.valkeyClaimName" -}}
{{- if .Values.valkey.persistence.existingClaim -}}
{{ .Values.valkey.persistence.existingClaim }}
{{- else -}}
{{ include "autoops.fullname" . }}-valkey
{{- end -}}
{{- end -}}
