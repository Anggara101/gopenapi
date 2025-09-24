package templates

type Model struct {
	Name         string
	OriginalName string
	Fields       []ModelProp
	Description  string
}

type ModelProp struct {
	GoName      string
	GoType      string
	JSONName    string
	Description string
}
