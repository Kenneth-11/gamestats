package gamestats

/** Mini JSON: parser + renderer con solo stdlib.
  * Soporta objetos, arreglos, strings (con escapes), números, bool, null.
  */
sealed trait J
case class JObj(fields: Map[String, J]) extends J
case class JArr(items: List[J]) extends J
case class JStr(value: String) extends J
case class JNum(value: Double) extends J
case class JBool(value: Boolean) extends J
case object JNull extends J

object Json:
  def parse(input: String): Either[String, J] =
    val p = Parser(input)
    try
      val v = p.parseValue()
      p.skipWs()
      if p.pos != input.length then Left(s"JSON: caracteres extra en pos ${p.pos}")
      else Right(v)
    catch case e: Exception => Left(s"JSON malformado: ${e.getMessage}")

  def render(j: J): String = j match
    case JNull       => "null"
    case JBool(b)    => if b then "true" else "false"
    case JNum(n)     => if n.isWhole then n.toLong.toString else n.toString
    case JStr(s)     => "\"" + escape(s) + "\""
    case JArr(items) => items.map(render).mkString("[", ",", "]")
    case JObj(fs)    => fs.map((k, v) => "\"" + escape(k) + "\":" + render(v)).mkString("{", ",", "}")

  private def escape(s: String): String =
    s.flatMap {
      case '"'  => "\\\""
      case '\\' => "\\\\"
      case '\n' => "\\n"
      case '\r' => "\\r"
      case '\t' => "\\t"
      case c if c < ' ' => f"\\u${c.toInt}%04x"
      case c    => c.toString
    }

  // Helpers de lectura
  def str(o: JObj, k: String): String = o.fields.get(k) match
    case Some(JStr(s)) => s
    case Some(JNum(n)) => if n.isWhole then n.toLong.toString else n.toString
    case _             => ""

  def obj(o: JObj, k: String): JObj = o.fields.get(k) match
    case Some(o2: JObj) => o2
    case _              => JObj(Map.empty)

  def arr(o: JObj, k: String): List[J] = o.fields.get(k) match
    case Some(JArr(xs)) => xs
    case _              => Nil

  def num(o: JObj, k: String, d: Double = 0.0): Double = o.fields.get(k) match
    case Some(JNum(n)) => n
    case Some(JStr(s)) => s.toDoubleOption.getOrElse(d)
    case _             => d

  private class Parser(input: String):
    var pos: Int = 0
    def skipWs(): Unit =
      while pos < input.length && input(pos).isWhitespace do pos += 1
    def parseValue(): J =
      skipWs()
      if pos >= input.length then throw new Exception("fin inesperado")
      input(pos) match
        case '{' => parseObj()
        case '[' => parseArr()
        case '"' => JStr(parseString())
        case 't' => expect("true"); JBool(true)
        case 'f' => expect("false"); JBool(false)
        case 'n' => expect("null"); JNull
        case c if c == '-' || c.isDigit => parseNum()
        case c => throw new Exception(s"carácter inesperado '$c' en $pos")
    def expect(s: String): Unit =
      if !input.startsWith(s, pos) then throw new Exception(s"se esperaba $s en $pos")
      pos += s.length
    def parseObj(): JObj =
      pos += 1 // {
      skipWs()
      var m = Map.empty[String, J]
      if pos < input.length && input(pos) == '}' then { pos += 1; return JObj(m) }
      while true do
        skipWs()
        val k = parseString()
        skipWs()
        if pos >= input.length || input(pos) != ':' then throw new Exception(s"se esperaba ':' en $pos")
        pos += 1
        m = m + (k -> parseValue())
        skipWs()
        if pos >= input.length then throw new Exception("objeto sin cerrar")
        if input(pos) == '}' then { pos += 1; return JObj(m) }
        if input(pos) != ',' then throw new Exception(s"se esperaba ',' en $pos")
        pos += 1
      JObj(m)
    def parseArr(): JArr =
      pos += 1 // [
      skipWs()
      var xs = List.empty[J]
      if pos < input.length && input(pos) == ']' then { pos += 1; return JArr(xs) }
      while true do
        xs = xs :+ parseValue()
        skipWs()
        if pos >= input.length then throw new Exception("arreglo sin cerrar")
        if input(pos) == ']' then { pos += 1; return JArr(xs) }
        if input(pos) != ',' then throw new Exception(s"se esperaba ',' en $pos")
        pos += 1
      JArr(xs)
    def parseString(): String =
      if input(pos) != '"' then throw new Exception(s"se esperaba '\"' en $pos")
      pos += 1
      val sb = new StringBuilder
      while pos < input.length && input(pos) != '"' do
        if input(pos) == '\\' then
          pos += 1
          if pos >= input.length then throw new Exception("escape incompleto")
          input(pos) match
            case '"'  => sb += '"'; pos += 1
            case '\\' => sb += '\\'; pos += 1
            case '/'  => sb += '/'; pos += 1
            case 'n'  => sb += '\n'; pos += 1
            case 'r'  => sb += '\r'; pos += 1
            case 't'  => sb += '\t'; pos += 1
            case 'u' =>
              if pos + 4 >= input.length then throw new Exception("unicode incompleto")
              sb += Integer.parseInt(input.substring(pos + 1, pos + 5), 16).toChar
              pos += 5
            case c => throw new Exception(s"escape inválido '\\$c'")
        else
          sb += input(pos); pos += 1
      if pos >= input.length then throw new Exception("string sin cerrar")
      pos += 1 // "
      sb.toString
    def parseNum(): JNum =
      val start = pos
      if pos < input.length && input(pos) == '-' then pos += 1
      while pos < input.length && (input(pos).isDigit || input(pos) == '.' || input(pos) == 'e' || input(pos) == 'E' || input(pos) == '+' || input(pos) == '-') do pos += 1
      JNum(input.substring(start, pos).toDoubleOption.getOrElse(throw new Exception(s"número inválido en $start")))
end Json
