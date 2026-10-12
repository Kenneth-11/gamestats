package gamestats

/** Modelo de eventos. Data queda como JObj para no perder campos específicos. */
case class Event(
  eventId: String,
  timestamp: String,
  gameId: String,
  gameVersion: String,
  matchId: String,
  playerId: String,
  eventType: String,
  action: String,
  data: JObj
)

case class MatchRequest(matchId: String, gameId: String, events: List[Event])

object Models:
  def eventFromJson(j: J): Either[String, Event] = j match
    case o: JObj =>
      val e = Event(
        Json.str(o, "eventId"),
        Json.str(o, "timestamp"),
        Json.str(o, "gameId"),
        Json.str(o, "gameVersion"),
        Json.str(o, "matchId"),
        Json.str(o, "playerId"),
        Json.str(o, "type"),
        Json.str(o, "action"),
        Json.obj(o, "data")
      )
      if e.eventId.isEmpty || e.matchId.isEmpty || e.gameId.isEmpty || e.eventType.isEmpty then
        Left("evento incompleto: eventId/matchId/gameId/type obligatorios")
      else Right(e)
    case _ => Left("evento debe ser objeto JSON")

  def requestFromJson(j: J): Either[String, MatchRequest] = j match
    case o: JObj =>
      val matchId = Json.str(o, "matchId")
      val gameId = Json.str(o, "gameId")
      if matchId.isEmpty || gameId.isEmpty then return Left("matchId/gameId obligatorios")
      val parsed: List[Either[String, Event]] = Json.arr(o, "events").map(eventFromJson)
      val errors = parsed.collect { case Left(err) => err }
      if errors.nonEmpty then Left(errors.mkString("; "))
      else Right(MatchRequest(matchId, gameId, parsed.collect { case Right(e) => e }))
    case _ => Left("body debe ser objeto {matchId, gameId, events}")
end Models
