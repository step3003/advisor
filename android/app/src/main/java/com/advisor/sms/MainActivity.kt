package com.advisor.sms

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Bundle
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import kotlin.concurrent.thread

/**
 * MainActivity: если не выполнен вход — показывает экран входа в аккаунт; иначе
 * показывает кабинет (WebView с нашим веб-UI), автоматически залогиненный.
 */
class MainActivity : AppCompatActivity() {
    private lateinit var prefs: Prefs

    private val requestSms =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        prefs = Prefs(this)
        if (prefs.isLoggedIn) showCabinet() else showLogin()
    }

    // --- Экран входа ---
    private fun showLogin() {
        setContentView(R.layout.activity_login)
        val urlField = findViewById<EditText>(R.id.urlField)
        val userField = findViewById<EditText>(R.id.userField)
        val passField = findViewById<EditText>(R.id.passField)
        val status = findViewById<TextView>(R.id.status)
        urlField.setText(prefs.serverUrl.ifEmpty { "https://" })

        findViewById<Button>(R.id.loginButton).setOnClickListener {
            val url = urlField.text.toString().trim().trimEnd('/')
            val user = userField.text.toString().trim()
            val pass = passField.text.toString()
            if (url.isEmpty() || user.isEmpty() || pass.isEmpty()) {
                status.text = "Заполните адрес, логин и пароль"
                return@setOnClickListener
            }
            status.text = "Вход…"
            thread {
                val token = Api.login(url, user, pass)
                runOnUiThread {
                    if (token != null) {
                        prefs.serverUrl = url
                        prefs.token = token
                        prefs.username = user
                        showCabinet()
                    } else {
                        status.text = "Не удалось войти: проверьте адрес, логин и пароль"
                    }
                }
            }
        }
    }

    // --- Кабинет (WebView) ---
    private fun showCabinet() {
        setContentView(R.layout.activity_web)
        ensureSmsPermission()

        val web = findViewById<WebView>(R.id.webview)
        web.settings.javaScriptEnabled = true
        web.settings.domStorageEnabled = true
        val token = prefs.token
        web.webViewClient = object : WebViewClient() {
            override fun onPageStarted(view: WebView, url: String?, favicon: android.graphics.Bitmap?) {
                // Прокидываем токен аккаунта в localStorage, чтобы WebView был залогинен.
                view.evaluateJavascript(
                    "try{localStorage.setItem('advisor.token','$token')}catch(e){}", null
                )
            }
        }
        web.loadUrl(prefs.serverUrl)

        findViewById<Button>(R.id.settingsButton).setOnClickListener {
            startActivity(Intent(this, SettingsActivity::class.java))
        }
    }

    private fun ensureSmsPermission() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.RECEIVE_SMS)
            != PackageManager.PERMISSION_GRANTED
        ) {
            requestSms.launch(Manifest.permission.RECEIVE_SMS)
        }
    }
}
