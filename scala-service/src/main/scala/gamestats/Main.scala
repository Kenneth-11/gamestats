package gamestats

import com.sun.net.httpserver.{HttpExchange, HttpHandler, HttpServer}
import java.net.InetSocketAddress
import java.nio.charset.StandardCharsets

/** Servidor HTTP con solo JDK (sin frameworks).
  * GET  /health  -> {"status":"ok","service":"gamestats-scala"}
  * POST /analyze -> {matchId, gameId, events:[...]} -> {status, statistics, patterns}
  */
object Main:
  def main(args: Array[String]): Unit =
    val port = sys.env.get("SCALA_SERVICE_PORT").flatMap(_.toIntOption).getOrElse(8081)
    val server = HttpServer.create(new InetSocketAddress(port), 0)
    server.createContext("/health", new HttpHandler:
      def handle(ex: HttpExchange): Unit =
        respond(ex, 200, """{"status":"ok","service":"gamestats-scala"}""")
    )
    server.createContext("/analyze", new HttpHandler:
      def handle(ex: HttpExchange): Unit =
        if ex.getRequestMethod != "POST" then
          respond(ex, 405, """{"status":"error","message":"solo POST"}""")
        else
          val body = new String(ex.getRequestBody.readAllBytes(), StandardCharsets.UTF_8)
          (for
            json <- Json.parse(body).left.map(identity)
            req  <- Models.requestFromJson(json)
          yield req) match
            case Left(err) => respond(ex, 422, s"""{"status":"error","message":${Json.render(JStr(err))}}""")
            case Right(req) =>
              val out = Json.render(Analytics.analyze(req))
              respond(ex, 200, out)
    )
    server.setExecutor(java.util.concurrent.Executors.newFixedThreadPool(4))
    server.start()
    println(s"gamestats-scala escuchando en :$port")

  private def respond(ex: HttpExchange, code: Int, body: String): Unit =
    val bytes = body.getBytes(StandardCharsets.UTF_8)
    ex.getResponseHeaders.set("Content-Type", "application/json; charset=utf-8")
    ex.sendResponseHeaders(code, bytes.length)
    val os = ex.getResponseBody
    try os.write(bytes) finally { os.close(); ex.close() }
end Main
