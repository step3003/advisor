package com.advisor.sms

import android.app.Notification
import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import kotlin.concurrent.thread

/**
 * NotificationService читает уведомления выбранных приложений (напр. банковских)
 * и пересылает их текст на сервер так же, как SMS. Требует у пользователя выдачи
 * «Доступа к уведомлениям». Ловит онлайн-оплаты, о которых банк шлёт пуш, а не SMS.
 */
class NotificationService : NotificationListenerService() {
    override fun onNotificationPosted(sbn: StatusBarNotification) {
        val prefs = Prefs(applicationContext)
        if (!prefs.captureNotif || !prefs.isLoggedIn) return

        val pkg = sbn.packageName ?: return
        // Фильтр приложений: слушаем только указанные (пусто = ничего, чтобы не спамить).
        val watch = prefs.notifPackages
        if (watch.isEmpty()) return
        val match = watch.split(",").map { it.trim().lowercase() }.filter { it.isNotEmpty() }
            .any { pkg.lowercase().contains(it) }
        if (!match) return

        val extras = sbn.notification?.extras ?: return
        val title = extras.getCharSequence(Notification.EXTRA_TITLE)?.toString().orEmpty()
        val text = (extras.getCharSequence(Notification.EXTRA_BIG_TEXT)
            ?: extras.getCharSequence(Notification.EXTRA_TEXT))?.toString().orEmpty()
        val full = listOf(title, text).filter { it.isNotBlank() }.joinToString(". ")
        if (full.isBlank()) return

        // Сеть — вне главного потока; сервис остаётся жив, отдельный pending не нужен.
        thread {
            // sender = package приложения; шаблоны на сервере матчат по нему и regex.
            Api.ingest(prefs, pkg, full)
        }
    }
}
