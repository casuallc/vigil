package inspection

type Request struct {
  Envs   map[string]string
  Config InspectionConfig
}
