plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.advisor.sms"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.advisor.sms"
        minSdk = 24
        targetSdk = 34
        versionCode = 2
        versionName = "1.1"
    }

    // Фиксированный ключ подписи (в репозитории). Гарантирует, что все сборки
    // подписаны одинаково — новые APK ставятся поверх старых без «конфликта».
    // Для сайдлоада это нормально; это не Play-релиз, пароль ничего ценного не защищает.
    signingConfigs {
        create("shared") {
            storeFile = file("advisor.jks")
            storePassword = "advisor123"
            keyAlias = "advisor"
            keyPassword = "advisor123"
        }
    }

    buildTypes {
        debug {
            signingConfig = signingConfigs.getByName("shared")
        }
        release {
            isMinifyEnabled = false
            signingConfig = signingConfigs.getByName("shared")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions {
        jvmTarget = "17"
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("androidx.activity:activity-ktx:1.9.2")
    implementation("com.google.android.material:material:1.12.0")
}
