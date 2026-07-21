package com.advisor.sms

import android.util.Log
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

/** Forwarder — отправка сырого SMS на сервер Advisor (POST /api/ingest/sms). */
object Forwarder {
    /**
     * send выполняет синхронный POST. Возвращает true при успехе (2xx).
     * Вызывать вне главного потока.
     */
    fun send(prefs: Prefs, sender: String, text: String): Boolean {
        if (!prefs.isConfigured) return false
        val body = JSONObject()
            .put("sender", sender)
            .put("text", text)
            .toString()

        return try {
            val url = URL(prefs.serverUrl + "/api/ingest/sms")
            val conn = url.openConnection() as HttpURLConnection
            conn.requestMethod = "POST"
            conn.connectTimeout = 10_000
            conn.readTimeout = 10_000
            conn.doOutput = true
            conn.setRequestProperty("Content-Type", "application/json")
            conn.setRequestProperty("Authorization", "Bearer " + prefs.token)
            conn.outputStream.use { it.write(body.toByteArray(Charsets.UTF_8)) }

            val code = conn.responseCode
            conn.disconnect()
            val ok = code in 200..299
            if (!ok) Log.w("AdvisorSMS", "сервер вернул $code")
            ok
        } catch (e: Exception) {
            Log.w("AdvisorSMS", "ошибка отправки: ${e.message}")
            false
        }
    }
}
