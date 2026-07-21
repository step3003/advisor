package com.advisor.sms

import android.content.Context

/** Prefs — хранилище настроек форвардера (URL сервера, токен, фильтр отправителя). */
class Prefs(context: Context) {
    private val sp = context.getSharedPreferences("advisor", Context.MODE_PRIVATE)

    var serverUrl: String
        get() = sp.getString("serverUrl", "") ?: ""
        set(v) = sp.edit().putString("serverUrl", v.trimEnd('/')).apply()

    var token: String
        get() = sp.getString("token", "") ?: ""
        set(v) = sp.edit().putString("token", v.trim()).apply()

    /** senderFilter — если не пусто, пересылаются только SMS, чей отправитель содержит эту подстроку. */
    var senderFilter: String
        get() = sp.getString("senderFilter", "") ?: ""
        set(v) = sp.edit().putString("senderFilter", v.trim()).apply()

    var enabled: Boolean
        get() = sp.getBoolean("enabled", false)
        set(v) = sp.edit().putBoolean("enabled", v).apply()

    val isConfigured: Boolean
        get() = serverUrl.isNotEmpty() && token.isNotEmpty()
}
