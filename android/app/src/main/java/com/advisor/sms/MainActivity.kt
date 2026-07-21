package com.advisor.sms

import android.Manifest
import android.content.pm.PackageManager
import android.os.Bundle
import android.widget.Button
import android.widget.EditText
import android.widget.Switch
import android.widget.TextView
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import kotlin.concurrent.thread

/** MainActivity — экран настройки форвардера: адрес сервера, токен, фильтр, вкл/выкл, тест. */
class MainActivity : AppCompatActivity() {
    private lateinit var prefs: Prefs
    private lateinit var urlField: EditText
    private lateinit var tokenField: EditText
    private lateinit var filterField: EditText
    private lateinit var enabledSwitch: Switch
    private lateinit var status: TextView

    private val requestSms =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
            status.text = if (granted) "Разрешение на SMS выдано" else "Без разрешения на SMS приём не работает"
        }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        prefs = Prefs(this)

        urlField = findViewById(R.id.urlField)
        tokenField = findViewById(R.id.tokenField)
        filterField = findViewById(R.id.filterField)
        enabledSwitch = findViewById(R.id.enabledSwitch)
        status = findViewById(R.id.status)

        urlField.setText(prefs.serverUrl)
        tokenField.setText(prefs.token)
        filterField.setText(prefs.senderFilter)
        enabledSwitch.isChecked = prefs.enabled

        findViewById<Button>(R.id.saveButton).setOnClickListener { save() }
        findViewById<Button>(R.id.testButton).setOnClickListener { test() }

        ensureSmsPermission()
    }

    private fun ensureSmsPermission() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.RECEIVE_SMS)
            != PackageManager.PERMISSION_GRANTED
        ) {
            requestSms.launch(Manifest.permission.RECEIVE_SMS)
        }
    }

    private fun save() {
        prefs.serverUrl = urlField.text.toString()
        prefs.token = tokenField.text.toString()
        prefs.senderFilter = filterField.text.toString()
        prefs.enabled = enabledSwitch.isChecked
        Toast.makeText(this, "Сохранено", Toast.LENGTH_SHORT).show()
        ensureSmsPermission()
    }

    /** test отправляет тестовое SMS на сервер, проверяя связку URL+токен. */
    private fun test() {
        save()
        status.text = "Отправка теста…"
        thread {
            val ok = Forwarder.send(prefs, "TEST", "Oplata 1.00 BYN test")
            runOnUiThread {
                status.text = if (ok) "Тест успешен: сервер принял SMS" else "Ошибка: проверьте URL и токен"
            }
        }
    }
}
