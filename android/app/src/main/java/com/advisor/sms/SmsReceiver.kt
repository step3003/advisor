package com.advisor.sms

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.provider.Telephony
import kotlin.concurrent.thread

/**
 * SmsReceiver ловит входящие SMS и пересылает сырой текст на сервер Advisor,
 * который разбирает его по настроенным в кабинете шаблонам. Само приложение
 * ничего не парсит — только форвардит.
 */
class SmsReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Telephony.Sms.Intents.SMS_RECEIVED_ACTION) return
        val prefs = Prefs(context)
        if (!prefs.enabled || !prefs.isConfigured) return

        // Собираем полный текст (SMS может прийти несколькими частями).
        val messages = Telephony.Sms.Intents.getMessagesFromIntent(intent) ?: return
        if (messages.isEmpty()) return
        val sender = messages[0].originatingAddress ?: ""
        val text = messages.joinToString("") { it.messageBody ?: "" }

        // Фильтр отправителя (если задан).
        val filter = prefs.senderFilter
        if (filter.isNotEmpty() && !sender.contains(filter, ignoreCase = true)) return

        // Сеть — вне главного потока; goAsync даёт время на завершение.
        val pending = goAsync()
        thread {
            try {
                Forwarder.send(prefs, sender, text)
            } finally {
                pending.finish()
            }
        }
    }
}
