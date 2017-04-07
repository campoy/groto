# Rules

Proto = Syntax { Import | Package | Option | Message | Enum | Service | EmptyStatement }


Message = message ident MessageBody
MessageBody = openBraces { Field | Enum | Message | Option | Oneof | MapField | Reserved | semicolon } closeBraces

MapField = map openAngled keyType* comma Type closeAngled ident equals intLit MaybeFieldOptions semicolon

Reserved = reserved ( Ranges | FieldNames ) semicolon
Ranges = Range { comma Range }
Range = ( intLit | intLit to intLit ) 
FieldNames = ident { comma ident }

Type = ident # types are a kind of ident

Constant = fullIdent | ( [ minus | plus ] intLit ) | ( [ minus | plus ] floatLit ) | strLit | boolLit 

Service = service ident openBrace { Option | RPC | semicolon } closeBrace
RPC = rpc ident RPCParam returns RPCParam ( RPCBody | semicolon )
RPCParam = openParens [ stream ] messageType closeParens
RPCBody = openBrace { Option | semicolon } closeBrace

IntLit = decimalLit | octalLit | hexLit

# Tokens

## Complex

strLit = ( "'" { charValue } "'" ) |  ( '"' { charValue } '"' )
fullIdent = ident { "." ident }
ident = letter { letter | decimalDigit | "_" }
dottedIdent =  [ "." ] { ident "." } ident

floatLit
decimalLit
octalLit
hexLit

## Keywords

enum
import
map
message
oneof
package
public
repeated
reserved
returns
rpc
service
stream
syntax
true
false
to
weak

### Types

#### Non Key Types

bytes
double
float

##### Key types

bool
fixed32
fixed64
int32
int64
sfixed32
sfixed64
sint32
sint64
string
uint32
uint64

## Punctuation

equals =
semicolon ;
quote = " '
openParens (
closeParens )
openBraces {
closeBraces }
openBrackets [
closeBracket ]
openAngled <
closeAngled >
comma ,
minus -
plus +


## Classes


keyType


## TODO

enumType and messageTypes can start with a '.'
This should be added to Type and RPCParam

## Notes to spec

ranges is not defined

comments are not defined

stream in service is not defined
    service = "service" serviceName "{" { option | rpc | stream | emptyStatement } "}"