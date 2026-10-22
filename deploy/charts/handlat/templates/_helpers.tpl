{{- define "handlat.fullname" -}}
{{- if .Release.IsUpgrade -}}
{{- if contains "laga" .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-laga" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- else if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "handlat.selectorLabels" -}}
app.kubernetes.io/name: {{ if .Release.IsUpgrade }}laga{{ else }}{{ .Chart.Name }}{{ end }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "handlat.labels" -}}
{{ include "handlat.selectorLabels" . }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "handlat.image" -}}
{{- $repository := required "image.repository is required" .Values.image.repository -}}
{{- if .Values.image.digest -}}
{{- if not (regexMatch "^sha256:[a-f0-9]{64}$" .Values.image.digest) -}}
{{- fail "image.digest must be a sha256 digest" -}}
{{- end -}}
{{- printf "%s@%s" $repository .Values.image.digest -}}
{{- else -}}
{{- $tag := required "set image.tag to a commit SHA or set image.digest" .Values.image.tag -}}
{{- if eq $tag "latest" -}}
{{- fail "use a commit SHA tag or digest instead of latest" -}}
{{- end -}}
{{- printf "%s:%s" $repository $tag -}}
{{- end -}}
{{- end -}}

{{- define "handlat.claimName" -}}
{{- default (include "handlat.fullname" .) .Values.persistence.existingClaim -}}
{{- end -}}
