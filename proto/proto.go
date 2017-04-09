package proto

// A File contains all the information that one can define in a .proto file.
type File struct {
	Syntax   Syntax
	Package  Package
	Imports  []Import
	Options  []Option
	Messages []Message
	Enums    []Enum
	Services []Service
}

// Syntax defines the protobuf version, it is always "proto3".
type Syntax struct{ Value string }

// An ImportModifier modifies an import statement to be weak or public.
type ImportModifier int

const (
	NormalImport ImportModifier = iota
	WeakImport
	PublicImport
)

// An Import statement is used to import definitions from other files.
type Import struct {
	Modifier ImportModifier
	Path     string
}

// A Package statement can be used to prevent name clashes between protocol message types.
type Package struct {
	Identifier []Identifier
}

// An Identifier is used as names for messages, services, options, etc.
type Identifier string

// An Option can be used in proto files, messages, enums and services.
// An option can be a protobuf defined option or a custom option.
// For more information, see Options in the language guide.
// https://developers.google.com/protocol-buffers/docs/proto3#options
type Option struct {
	Prefix []Identifier // Parenthesised part of the identifier, if any.
	Name   []Identifier
	Value  interface{}
}

// A Message consists of a message name and a message body.
// The message body can have fields, nested enum definitions,
// nested message definitions, options, oneofs, map fields,
// and reserved statements.
type Message struct {
	Name      Identifier
	Fields    []Field
	Enums     []Enum
	Messages  []Message
	Options   []Option
	OneOfs    []OneOf
	Maps      []Map
	Reserveds []Reserved
}

// Fields are the basic elements of a protocol buffer message.
type Field struct {
	Repeated bool
	Type     Type
	Name     Identifier
	Number   int
	Options  []Option
}

// An Enum consists of a name and an enum body.
// The enum body can have options and enum fields.
type Enum struct {
	Name    Identifier
	Fields  []EnumField
	Options []Option
}

// An EnumField is one of the values defined in an Enum.
type EnumField struct {
	Name    Identifier
	Number  int
	Options []Option
}

// A OneOf provides a way to define when only one of a set of fields
// can be set at any time.
type OneOf struct {
	Name   Identifier
	Fields []OneOfField
}

// A OneOfField is one of the possible fields in a OneOf statement.
type OneOfField struct {
	Type    Type
	Name    Identifier
	Number  int
	Options []Option
}

// A Map field has a key type, value type, name, and field number.
// The key type can be any integral or string type.
type Map struct {
	KeyType   Type
	ValueType Type
	Name      Identifier
	Number    int
	Options   []Option
}

// Type contains either a predefined type in the form a Token,
// or a full identifier.
type Type struct {
	Predefined  PredefinedType
	UserDefined []Identifier
}

// A PredefinedType is a type that is part of the definition of the
// Protocol Buffer V3 language.
type PredefinedType int

const (
	TypeInvalid PredefinedType = iota
	TypeBytes
	TypeDouble
	TypeFloat
	TypeBool
	TypeFixed32
	TypeFixed64
	TypeInt32
	TypeInt64
	TypeSfixed32
	TypeSfixed64
	TypeSint32
	TypeSint64
	TypeString
	TypeUint32
	TypeUint64
)

// A Reserved statement declares a range of field numbers or field
// names that cannot be used in this message.
type Reserved struct {
	IDs    []int
	Names  []string
	Ranges []Range
}

// A Range defines a range of values that are reserved in a Reserved statement.
type Range struct{ From, To int }

// A Service is defined by its name and a list of RPC methods.
type Service struct {
	Name    Identifier
	Options []Option
	RPCs    []RPC
}

// A RPC method defines a remote procedure call with a name,
// input and output types, and options.
type RPC struct {
	Name    Identifier
	In      RPCParam
	Out     RPCParam
	Options []Option
}

// An RPCParam defines an input or output parameter for an RPC service.
type RPCParam struct {
	Stream bool
	Type   []Identifier
}
