package com.advisor.sms

import android.content.Context

/** Prefs — настройки приложения: сервер, токен аккаунта, что захватывать, фильтры. */
class Prefs(context: Context) {
    private val sp = context.getSharedPreferences("advisor", Context.MODE_PRIVATE)

    var serverUrl: String
        get() = sp.getString("serverUrl", "") ?: ""
        set(v) = sp.edit().putString("serverUrl", v.trim().trimEnd('/')).apply()

    /** token — токен сессии аккаунта (получен при входе), используется и WebView, и захватом. */
    var token: String
        get() = sp.getString("token", "") ?: ""
        set(v) = sp.edit().putString("token", v.trim()).apply()

    var username: String
        get() = sp.getString("username", "") ?: ""
        set(v) = sp.edit().putString("username", v.trim()).apply()

    /** captureSms — пересылать входящие SMS. */
    var captureSms: Boolean
        get() = sp.getBoolean("captureSms", true)
        set(v) = sp.edit().putBoolean("captureSms", v).apply()

    /** captureNotif — пересылать уведомления выбранных приложений. */
    var captureNotif: Boolean
        get() = sp.getBoolean("captureNotif", false)
        set(v) = sp.edit().putBoolean("captureNotif", v).apply()

    /** smsSenderFilter — подстрока отправителя SMS (пусто = любой). */
    var smsSenderFilter: String
        get() = sp.getString("smsSenderFilter", "") ?: ""
        set(v) = sp.edit().putString("smsSenderFilter", v.trim()).apply()

    /**
     * notifPackages — какие приложения слушать (подстроки package через запятую,
     * напр. "priorbank,alfa"). Пусто = не слать ничего (чтобы не заваливать сервер).
     */
    var notifPackages: String
        get() = sp.getString("notifPackages", "") ?: ""
        set(v) = sp.edit().putString("notifPackages", v.trim()).apply()

    val isLoggedIn: Boolean
        get() = serverUrl.isNotEmpty() && token.isNotEmpty()

    fun clearSession() {
        sp.edit().remove("token").apply()
    }
}
