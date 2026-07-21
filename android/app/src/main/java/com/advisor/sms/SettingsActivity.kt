package com.advisor.sms

import android.content.Intent
import android.os.Bundle
import android.provider.Settings
import android.widget.Button
import android.widget.EditText
import android.widget.Switch
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity

/** SettingsActivity — что захватывать (SMS/уведомления), фильтры, доступ к уведомлениям, выход. */
class SettingsActivity : AppCompatActivity() {
    private lateinit var prefs: Prefs

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        prefs = Prefs(this)
        setContentView(R.layout.activity_settings)

        val smsSwitch = findViewById<Switch>(R.id.smsSwitch)
        val notifSwitch = findViewById<Switch>(R.id.notifSwitch)
        val smsFilter = findViewById<EditText>(R.id.smsFilter)
        val notifPackages = findViewById<EditText>(R.id.notifPackages)

        smsSwitch.isChecked = prefs.captureSms
        notifSwitch.isChecked = prefs.captureNotif
        smsFilter.setText(prefs.smsSenderFilter)
        notifPackages.setText(prefs.notifPackages)

        findViewById<Button>(R.id.notifAccessButton).setOnClickListener {
            // Экран системного разрешения «Доступ к уведомлениям».
            startActivity(Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS))
        }

        findViewById<Button>(R.id.saveButton).setOnClickListener {
            prefs.captureSms = smsSwitch.isChecked
            prefs.captureNotif = notifSwitch.isChecked
            prefs.smsSenderFilter = smsFilter.text.toString()
            prefs.notifPackages = notifPackages.text.toString()
            Toast.makeText(this, "Сохранено", Toast.LENGTH_SHORT).show()
            finish()
        }

        findViewById<Button>(R.id.logoutButton).setOnClickListener {
            prefs.clearSession()
            val i = Intent(this, MainActivity::class.java)
            i.flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            startActivity(i)
            finish()
        }
    }
}
