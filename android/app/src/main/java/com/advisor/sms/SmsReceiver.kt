package com.advisor.sms

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.provider.Telephony
import kotlin.concurrent.thread

/**
 * SmsReceiver ловит входящие SMS и пересылает сырой текст на сервер (если включён
 * захват SMS и выполнен вход). Разбор — на сервере по шаблонам.
 */
class SmsReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Telephony.Sms.Intents.SMS_RECEIVED_ACTION) return
        val prefs = Prefs(context)
        if (!prefs.captureSms || !prefs.isLoggedIn) return

        val messages = Telephony.Sms.Intents.getMessagesFromIntent(intent) ?: return
        if (messages.isEmpty()) return
        val sender = messages[0].originatingAddress ?: ""
        val text = messages.joinToString("") { it.messageBody ?: "" }

        val filter = prefs.smsSenderFilter
        if (filter.isNotEmpty() && !sender.contains(filter, ignoreCase = true)) return

        val pending = goAsync()
        thread {
            try {
                Api.ingest(prefs, sender, text)
            } finally {
                pending.finish()
            }
        }
    }
}
