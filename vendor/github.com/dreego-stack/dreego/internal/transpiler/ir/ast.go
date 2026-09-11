package ir

type TemplateNodeType int

const (
	NodeText TemplateNodeType = iota
	NodeExpression
	NodeMessage
	NodeIf
	NodeEach
	NodeSlot
	NodeComponentCall
	NodeVerbatim
	NodeClientScript
)

type TemplateNode struct {
	Type         TemplateNodeType
	Content      string
	Cond         string
	Items        string
	Item         string
	Children     []TemplateNode
	ElseChildren []TemplateNode
	Tag          string
	Attrs        string
	SelfClose    bool
	Filters      []string
	Pos          int
	Source       string
	SourceText   string
	Language     string
	MessageKey   string
	MessageArgs  []MessageArgument
}

type MessageArgument struct {
	Name       string
	Expression string
}

type ServerSection struct {
	Code           string
	Language       string
	Method         string
	MethodExplicit bool
	ContentType    string
	Action         string
}

type File struct {
	Head          *HeadSection
	Server        []ServerSection
	Body          *BodySection
	Bodies        []BodySection
	Client        *ClientSection
	Style         *StyleSection
	Component     *ComponentDef
	Imports       []Import
	FormActions   []string
	SourceContent string
	SourcePath    string
}

type ComponentDef struct {
	Name           string
	Props          []Prop
	Slots          []string
	HasDefaultSlot bool
	HasNamedSlot   bool
}

type Prop struct {
	Name    string
	Type    string
	Default string
}

type Import struct {
	Alias string
	Path  string
	Names []string
}

type HeadSection struct {
	Content  string
	Language string
}

type BodySection struct {
	Nodes          []TemplateNode
	Language       string
	Method         string
	MethodExplicit bool
}

type ClientSection struct {
	Code     string
	Language string
	Pos      int
}

type StyleSection struct {
	Code     string
	Language string
}
