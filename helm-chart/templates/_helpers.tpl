{{/* Standard labels applied to every rendered object. */}}
{{- define "gitops-demo.labels" -}}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/part-of: gitops-eks-microservices-demo
app.kubernetes.io/managed-by: {{ .root.Release.Service }}
helm.sh/chart: {{ .root.Chart.Name }}-{{ .root.Chart.Version }}
{{- end -}}
