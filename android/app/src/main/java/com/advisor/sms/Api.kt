package com.advisor.sms

import android.util.Log
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

/** Api — сетевое взаимодействие с сервером Advisor. Вызывать вне главного потока. */
object Api {
    /** login выполняет вход в аккаунт и возвращает токен сессии, либо null. */
    fun login(serverUrl: String, username: String, password: String): String? {
        val body = JSONObject()
            .put("username", username)
            .put("password", password)
            .put("deviceName", "android")
            .toString()
        val (code, resp) = post("$serverUrl/api/auth/login", body, null)
        if (code !in 200..299 || resp == null) return null
        return try {
            JSONObject(resp).optString("token").ifEmpty { null }
        } catch (e: Exception) {
            null
        }
    }

    /**
     * ingest пересылает сырое сообщение (SMS или уведомление) на сервер, где оно
     * разбирается по шаблонам. Возвращает true при успехе.
     */
    fun ingest(prefs: Prefs, sender: String, text: String): Boolean {
        if (!prefs.isLoggedIn) return false
        val body = JSONObject()
            .put("sender", sender)
            .put("text", text)
            .toString()
        val (code, _) = post("${prefs.serverUrl}/api/ingest/sms", body, prefs.token)
        val ok = code in 200..299
        if (!ok) Log.w("Advisor", "ingest вернул $code")
        return ok
    }

    private fun post(urlStr: String, body: String, token: String?): Pair<Int, String?> {
        return try {
            val conn = URL(urlStr).openConnection() as HttpURLConnection
            conn.requestMethod = "POST"
            conn.connectTimeout = 10_000
            conn.readTimeout = 10_000
            conn.doOutput = true
            conn.setRequestProperty("Content-Type", "application/json")
            if (token != null) conn.setRequestProperty("Authorization", "Bearer $token")
            conn.outputStream.use { it.write(body.toByteArray(Charsets.UTF_8)) }
            val code = conn.responseCode
            val resp = try {
                (if (code in 200..299) conn.inputStream else conn.errorStream)
                    ?.bufferedReader()?.readText()
            } catch (e: Exception) {
                null
            }
            conn.disconnect()
            code to resp
        } catch (e: Exception) {
            Log.w("Advisor", "сеть: ${e.message}")
            -1 to null
        }
    }
}
