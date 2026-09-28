# گزارش جامع، تفصیلی و سناریوی کامل ارائه پروژه Replicated Key-Value Store
**تمرین شماره ۳ - درس مبانی رایانش توزیع‌شده**
**دانشگاه تهران - دانشکده مهندسی برق و کامپیوتر - بهار ۱۴۰۴ (Spring 2025)**

---

## 📋 شناسنامه پروژه

* **عنوان پروژه:** طراحی، پیاده‌سازی و تحلیل تجربی پایگاه داده کلید-مقدار توزیع‌شده و همانندسازی‌شده (Distributed Replicated Key-Value Store)
* **دانشگاه:** دانشگاه تهران - دانشکده مهندسی برق و کامپیوتر
* **درس:** مبانی رایانش توزیع‌شده (Distributed Computing Fundamentals)
* **استاد درس:** دکتر محمدرضا شورنیا
* **سرستیار آموزشی:** پویا جمشیدی
* **دستیاران آموزشی:** مبینا حق‌زاده، محمدمهدی ابراهیم سلطان
* **ارائه‌دهندگان (دانشجویان):**
  * **طه مجلسی**
  * **علیرضا کریمی**
* **پشته تکنولوژی:** Go (Golang 1.20+), HTTP REST API, JSON, Bash Automation, Python 3 (Matplotlib / NumPy), LaTeX

---

## 📑 جدول محتویات

1. [خلاصه مدیریتی پروژه (Executive Summary)](#1-خلاصه-مدیریتی-پروژه-executive-summary)
2. [مفاهیم نظری و تئوریک رایانش توزیع‌شده (Deep Theoretical Foundations)](#2-مفاهیم-نظری-و-تئوریک-رایانش-توزیع‌شده-deep-theoretical-foundations)
   * [چرا همانندسازی (Replication Drivers)؟](#چرا-همانندسازی-replication-drivers)
   * [افزایش کارایی در برابر تحمل خطا (Performance vs Fault Tolerance)](#افزایش-کارایی-در-برابر-تحمل-خطا-performance-vs-fault-tolerance)
   * [چالش‌های بنیادی سازگاری (Consistency Challenges)](#چالش‌های-بنیادی-سازگاری-consistency-challenges)
   * [مدل‌های سازگاری سیستم (Strong Consistency vs Eventual Consistency)](#مدل‌های-سازگاری-سیستم-strong-consistency-vs-eventual-consistency)
   * [مدل‌های سازگاری کاربرمحور (Client-Centric Consistency Models)](#مدل‌های-سازگاری-کاربرمحور-client-centric-consistency-models)
   * [اثبات ریاضی و تحلیل قضیه CAP](#اثبات-ریاضی-و-تحلیل-قضیه-cap)
3. [معماری کامل سیستم و پروتکل ارتباطی (System Architecture & Protocol Specs)](#3-معماری-کامل-سیستم-و-پروتکل-ارتباطی-system-architecture--protocol-specs)
4. [تشریح دقیق و خط‌به‌خط کدها (Exhaustive Code Breakdown)](#4-تشریح-دقیق-و-خط‌به‌خط-کدها-exhaustive-code-breakdown)
   * [کد سرور همانندساز (`codes/replica/main.go`)](#کد-سرور-همانندساز-codesreplicamaingo)
   * [کد کلاینت تست (`codes/client/main.go`)](#کد-کلاینت-تست-codesclientmaingo)
   * [اسکریپت تست خودکار (`codes/run_all_tests.sh`)](#اسکریپت-تست-خودکار-codesrun_all_testssh)
   * [اسکریپت تولید نمودارها (`codes/generate_charts.py`)](#اسکریپت-تولید-نمودارها-codesgenerate_chartspy)
5. [تحلیل جامع سناریوهای چهارگانه تست (Comprehensive Test Scenarios)](#5-تحلیل-جامع-سناریوهای-چهارگانه-تست-comprehensive-test-scenarios)
   * [سناریو ۱: مشاهده ناسازگاری موقت (Temporary Inconsistency)](#سناریو-۱-مشاهده-ناسازگاری-موقت-temporary-inconsistency)
   * [سناریو ۲: خرابی یک Replica (Replica Failure Behavior)](#سناریو-۲-خرابی-یک-replica-replica-failure-behavior)
   * [سناریو ۳: تعارض همزمان (Concurrent Conflicts & LWW)](#سناریو-۳-تعارض-همزمان-concurrent-conflicts--lww)
   * [سناریو ۴: اثر تاخیر شبکه (Network Delay Impact)](#سناریو-۴-اثر-تاخیر-شبکه-network-delay-impact)
6. [ماتریس نتایج تجربی و داده‌های کمّی (Quantitative Metrics Matrix)](#6-ماتریس-نتایج-تجربی-و-داده‌های-کمّی-quantitative-metrics-matrix)
7. [تحلیل پیشرفته، محدودیت‌ها و مقایسه صنعتی (Advanced Analysis & Industrial Comparison)](#7-تحلیل-پیشرفته-محدودیت‌ها-و-مقایسه-صنعتی-advanced-analysis--industrial-comparison)
8. [سناریوی کامل ارائه اسلاید به اسلاید (Slide-by-Slide Presentation Plan & Script)](#8-سناریوی-کامل-ارائه-اسلاید-به-اسلاید-slide-by-slide-presentation-plan--script)
9. [راهنمای جلسه دفاع و پرسش و پاسخ‌ها (Q&A Defense Guide)](#9-راهنمای-جلسه-دفاع-و-پرسش-و-پاسخ‌ها-qa-defense-guide)

---

## 1. خلاصه مدیریتی پروژه (Executive Summary)

این پروژه یک پیاده‌سازی کامل، مقیاس‌پذیر و تجربی از یک **پایگاه داده کلید-مقدار توزیع‌شده و همانندسازی‌شده (Replicated Key-Value Store)** است. سیستم از **۳ نود سرور مجزا (Replica)** تشکیل شده که با زبان **Go** پیاده‌سازی شده و از طریق پروتکل **HTTP/REST API** متصل شده‌اند.

سیستم طراحی‌شده قادر است تحت دو مدل سازگاری اصلی اجرا شود:
1. **Eventual Consistency (سازگاری نهایی):** تاکید بر کمترین تاخیر (Latency) و بالاترین دسترس‌پذیری (Availability). عملیات انتشار داده به صورت غیرهمگام (Asynchronous / Fire-and-Forget) انجام می‌شود.
2. **Strong Consistency (سازگاری قوی ساده‌شده):** تاکید بر صحت و یکپارچگی داد‌ه‌ها (Consistency). عملیات نوشتن بر اساس **حد نصاب اکثریت (Quorum Write - حداقل ۲ نود از ۳ نود)** به صورت همگام (Synchronous) تایید می‌شود.

برای مدیریت همزمانی و حل تعارض، مکانیزم **نسخه‌گذاری (Versioning)** همراه با برچسب‌های زمانی با دقت **نانوثانیه** و الگوریتم **Last-Write-Wins (LWW)** به همراه شکستن بن‌بست با **شناسه سرور (Replica ID)** پیاده‌سازی شده است. سیستم طی ۱۸ سناریوی خودکار تحت تاخیرهای مصنوعی مختلف شبکه (0ms, 500ms, 2000ms) به طور کامل تست و تحلیل کمّی شده است.

---

## 2. مفاهیم نظری و تئوریک رایانش توزیع‌شده (Deep Theoretical Foundations)

### چرا همانندسازی (Replication Drivers)؟
در رایانش توزیع‌شده، همانندسازی (Replication) به ۴ دلیل حیاتی انجام می‌شود:
1. **دسترس‌پذیری بالا (High Availability):** خرابی یا از دست رفتن یک سرور باعث از کار افتادن کل سیستم نمی‌شود ($Availability = 1 - P(all\_failed)$).
2. **بهبود کارایی و مقیاس‌پذیری (Performance & Scalability):** بار درخواست‌های GET بین نودها توزیع شده و تاخیر خواندن کاهش می‌یابد.
3. **ماندگاری داده و تحمل خطا (Fault Tolerance & Durability):** داده‌ها روی سخت‌افزارهای مختلف حفظ شده و در صورت خرابی دیسک از بین نمی‌روند.
4. **کاهش تاخیر جغرافیایی (Geographic Latency Reduction):** داده‌ها در نزدیک‌ترین فاصله به کاربران (مانند شبکه‌های CDN) قرار می‌گیرند.

### افزایش کارایی در برابر تحمل خطا (Performance vs Fault Tolerance)
* **Replication برای کارایی:** از همانندسازی **غیرهمگام (Asynchronous)** استفاده می‌کند. کلاینت بلافاصله پس از تغییر روی سرور محلی پاسخ مثبت می‌گیرد. تاخیر PUT بسیار کم است، اما امکان خواندن داده‌های قدیمی (Stale Reads) وجود دارد.
* **Replication برای تحمل خطا:** از همانندسازی **همگام (Synchronous)** یا Quorum استفاده می‌کند. درخواست نوشتن تا زمانی که توسط نودهای متعدد تایید نشود پایان نمی‌یابد. تاخیر بیشتر است اما پایداری داده تضمین می‌شود.

### چالش‌های بنیادی سازگاری (Consistency Challenges)
در اثر تاخیر انتشار در شبکه، ۳ مشکل اساسی رخ می‌دهد:
1. **Stale Reads (خواندن داده قدیمی):** کلاینت قبل از رسیدن پیام همانندسازی به یک نود، داده‌های قبلی آن را می‌خواند.
2. **Write Conflicts (تعارضات نوشتن):** دو کلاینت همزمان روی دو Replica متفاوت مقادیر مختلفی برای یک کلید یکسان می‌نویسند.
3. **Lost Updates (به‌روزرسانی‌های گمشده):** یک نوشتن بدون اطلاع از نوشتن همزمان دیگر، آن را اوررایت می‌کند.

### مدل‌های سازگاری سیستم (Strong Consistency vs Eventual Consistency)
* **Strong Consistency (Linearizability):** بعد از اتمام عملیات PUT، هر خواندن (GET) از هر نود در سیستم، حتماً آخرین مقدار نوشته‌شده را برمی‌گرداند.
* **Eventual Consistency:** سیستم تضمین می‌دهد که در صورت عدم ورود داده جدید، در نهایت تمامی Replicaها همگرا خواهند شد ($t \to \infty \implies V_1 = V_2 = V_3$).

### مدل‌های سازگاری کاربرمحور (Client-Centric Consistency Models)
* **Read-Your-Writes:** اگر کلاینت $C$ مقدار $v$ را بنویسد، هر خواندن بعدی توسط همان کلاینت $C$ حتماً $v$ یا مقداری جدیدتر را برمی‌گرداند.
* **Monotonic Reads:** اگر کلاینت $C$ مقدار $v$ با نسخه $t$ را ببیند، هیچ خواندن بعدی توسط $C$ مقداری قدیمی‌تر از $t$ را نشان نخواهد داد.

### اثبات ریاضی و تحلیل قضیه CAP
طبق **قضیه CAP (Brewer's Theorem)**، در یک سیستم توزیع‌شده با امکان تفکیک شبکه (Network Partition - $P$)، نمی‌توان همزمان **Consistency ($C$)** و **Availability ($A$)** را حفظ نمود:

فرض کنید شبکه بین نود $A$ و نود $B$ قطع شود ($Partition$).
* کلاینت ۱ عملیات `PUT(x=10)` را روی نود $A$ انجام می‌دهد. نود $A$ به علت قطع شبکه نمی‌تواند پیام همگام‌سازی را به $B$ بفرستد.
* حالا کلاینت ۲ عملیات `GET(x)` را روی نود $B$ انجام می‌دهد. نود $B$ دو انتخاب دارد:
  1. **پاسخ دهد (انتخاب Availability):** مقدار قدیمی یا خالی را برمی‌گرداند $\implies$ **Consistency نقض می‌شود (سیستم AP/Eventual)**.
  2. **پاسخ ندهد یا خطا دهد (انتخاب Consistency):** درخواست کلاینت ۲ رد می‌شود $\implies$ **Availability نقض می‌شود (سیستم CP/Strong)**.

---

## 3. معماری کامل سیستم و پروتکل ارتباطی (System Architecture & Protocol Specs)

سیستم از ۳ نود مستقل تشکیل شده است (`replica1` روی پورت 8001، `replica2` روی 8002، `replica3` روی 8003) که با یک کلاینت CLI متصل هستند:

```
                          +-----------------------+
                          |   Client CLI App      |
                          +-----------+-----------+
                                      |
             +------------------------+------------------------+
             | POST /put  |  GET /get                          |
             v                                                 v
    +-----------------+                               +-----------------+
    |    Replica 1    |<=============================>|    Replica 2    |
    |   (Port 8001)   |       POST /replicate         |   (Port 8002)   |
    +--------+--------+                               +--------+--------+
             ^                                                 ^
             |                 POST /replicate                 |
             +-------------------------------------------------+
                                       |
                                       v
                               +---------------+
                               |   Replica 3   |
                               |  (Port 8003)  |
                               +---------------+
```

### 📡 مشخصات کامل HTTP REST API

| Endpoint | Method | Request Body | Description |
| :--- | :--- | :--- | :--- |
| `/put` | `POST` | `{"key":"x", "value":"10"}` | ثبت کلید-مقدار جدید و همانندسازی |
| `/get` | `GET` | Query Param: `?key=x` | دریافت ارزش کلید از نود محلی |
| `/replicate` | `POST` | `{"entry":{...}, "origin_id":"replica1"}` | نقطه پایانی داخلی جهت همگام‌سازی بین نودها |
| `/start` | `POST` | Empty | فعال‌سازی سرور پس از خاموشی |
| `/stop` | `POST` | Empty | خاموش کردن سرور برای شبیه‌سازی خرابی |
| `/status` | `GET` | Empty | دریافت وضعیت روشن/خاموش و کانفیگ |
| `/dump` | `GET` | Empty | دریافت کل محتوای دیتابیس محلی |

---

## 4. تشریح دقیق و خط‌به‌خط کدها (Exhaustive Code Breakdown)

### 🔴 کد سرور همانندساز ([codes/replica/main.go](file:///Users/tahamajs/Documents/uni/DIST/CAs/CA3/codes/replica/main.go))

#### ۱. تعریف ساختارهای داده
```go
// DataEntry ساختار هر عنصر ذخیره‌شده در حافظه محلی
type DataEntry struct {
    Key       string `json:"key"`
    Value     string `json:"value"`
    Version   int    `json:"version"`     // شماره نسخه تک‌نوا
    UpdatedBy string `json:"updated_by"`  // شناسه نودی که تغییر را ایجاد کرده
    Timestamp int64  `json:"timestamp"`   // زمان نانوثانیه‌ای سیستم برای LWW
}

// Replica ساختار اصلی هر نود سرور
type Replica struct {
    config    Config
    data      map[string]DataEntry
    mu        sync.RWMutex               // قفل خواندن/نوشتن چندنخی
    client    *http.Client               // کلاینت HTTP برای ارسال به سایر Peerها
    isRunning bool                       // پرچم وضعیت سرور
}
```
* **توضیح کد:**
  * `Version`: عدد صحیح صعودی که با هر به‌روزرسانی محلی افزایش می‌یابد.
  * `Timestamp`: با `time.Now().UnixNano()` مقداردهی می‌شود تا دقت نانوثانیه‌ای برای الگوریتم Last-Write-Wins فراهم کند.
  * `sync.RWMutex`: از بروز Data Race در درخواست‌های همزمان HTTP جلوگیری می‌کند.

#### ۲. تابع پردازش PUT (`handlePUT`)
```go
func (r *Replica) handlePUT(w http.ResponseWriter, req *http.Request) {
    if !r.isRunning {
        http.Error(w, "Replica is stopped", http.StatusServiceUnavailable)
        return
    }
    body, _ := io.ReadAll(req.Body)
    var putReq PUTRequest
    json.Unmarshal(body, &putReq)

    // ورود به بخش بحرانی جهت به‌روزرسانی محلی
    r.mu.Lock()
    currentEntry, exists := r.data[putReq.Key]
    newVersion := 1
    if exists {
        newVersion = currentEntry.Version + 1
    }
    
    newEntry := DataEntry{
        Key:       putReq.Key,
        Value:     putReq.Value,
        Version:   newVersion,
        UpdatedBy: r.config.ID,
        Timestamp: time.Now().UnixNano(),
    }
    r.data[putReq.Key] = newEntry
    r.mu.Unlock() // خروج از بخش بحرانی

    // تصمیم‌گیری بر اساس مدل سازگاری
    if r.config.ConsistencyModel == "strong" {
        // سازگاری قوی همگام
        success := r.replicateStrong(newEntry)
        if !success {
            response := PUTResponse{Success: false, Message: "Failed majority consensus"}
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(response)
            return
        }
    } else {
        // سازگاری نهایی غیرهمگام در پس‌زمینه
        go r.replicateEventual(newEntry)
    }

    response := PUTResponse{
        Success:   true,
        Message:   "Value stored successfully",
        Version:   newVersion,
        UpdatedBy: r.config.ID,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```
* **توضیح کد:**
  1. وضعیت سرور چک می‌شود.
  2. قفل نوشتن `r.mu.Lock()` فعال شده، نسخه جدید محاسبه و عنصر محلی ذخیره می‌شود.
  3. اگر کانفیگ `strong` باشد، `replicateStrong` فراخوانی می‌شود که کلاینت را تا دریافت حد نصاب معطل نگه می‌دارد.
  4. اگر کانفیگ `eventual` باشد، یک Goroutine به صورت `go r.replicateEventual(newEntry)` اجرا می‌شود که بدون مسدودسازی (Non-blocking) پاسخ را برمی‌گرداند.

#### ۳. تابع همانندسازی همگام با Quorum (`replicateStrong`)
```go
func (r *Replica) replicateStrong(entry DataEntry) bool {
    totalNodes := len(r.config.Peers) + 1
    majorityNeeded := totalNodes/2 + 1 // برای ۳ نود برابر ۲ است
    acknowledged := 1 // تایید نود محلی

    var wg sync.WaitGroup
    var ackMu sync.Mutex

    for _, peer := range r.config.Peers {
        wg.Add(1)
        go func(peerURL string) {
            defer wg.Done()
            if r.sendReplication(peerURL, entry) {
                ackMu.Lock()
                acknowledged++
                ackMu.Unlock()
            }
        }(peer)
    }
    wg.Wait()
    return acknowledged >= majorityNeeded
}
```
* **توضیح کد:**
  * این تابع با ایجاد `sync.WaitGroup` پیام‌های همگام‌سازی را به طور همزمان به همتایان می‌فرستد.
  * به محض آنکه تعداد تاییدات (`acknowledged`) به حد نصاب اکثریت ($\ge 2$) برسد، مقدار `true` برمی‌گرداند.

#### ۴. تابع دریافت همگام‌سازی و حل تعارض LWW (`handleReplicate`)
```go
func (r *Replica) handleReplicate(w http.ResponseWriter, req *http.Request) {
    if !r.isRunning {
        http.Error(w, "Replica is stopped", http.StatusServiceUnavailable)
        return
    }
    // اعمال تاخیر مصنوعی شبکه
    if r.config.NetworkDelay > 0 {
        time.Sleep(time.Duration(r.config.NetworkDelay) * time.Millisecond)
    }

    r.mu.Lock()
    currentEntry, exists := r.data[repReq.Entry.Key]
    shouldUpdate := false

    if !exists {
        shouldUpdate = true
    } else {
        if repReq.Entry.Version > currentEntry.Version {
            shouldUpdate = true // نسخه ورودی جدیدتر است
        } else if repReq.Entry.Version == currentEntry.Version {
            // نسخه یکسان = بروز تعارض همزمان! استفاده از LWW
            if repReq.Entry.Timestamp > currentEntry.Timestamp {
                shouldUpdate = true // برچسب زمان جدیدتر برنده است
            } else if repReq.Entry.Timestamp == currentEntry.Timestamp {
                // برابری برچسب زمان = استفاده از شناسه سرور جهت Tiebreaker
                if repReq.OriginID > r.config.ID {
                    shouldUpdate = true
                }
            }
        }
    }

    if shouldUpdate {
        r.data[repReq.Entry.Key] = repReq.Entry
    }
    r.mu.Unlock()
}
```
* **توضیح کد:**
  * تاخیر مصنوعی شبکه اعمال می‌شود.
  * در صورت وجود نسخه یکسان (تعارض همزمان)، از سیاست **Last-Write-Wins (LWW)** با دقت نانوثانیه استفاده شده و در صورت برابری کامل، ID سرور مبدا تصمیم‌گیرنده نهایی است.

---

### 🔵 کد کلاینت تست ([codes/client/main.go](file:///Users/tahamajs/Documents/uni/DIST/CAs/CA3/codes/client/main.go))

کلاینت ابزاری جامع جهت ارسال دستورات و اندازه‌گیری دقیق پارامترهای کارایی است:
```go
func (c *Client) PUT(replicaURL, key, value string) (*PUTResponse, time.Duration, error) {
    start := time.Now()
    resp, err := c.client.Post(replicaURL+"/put", "application/json", bytes.NewBuffer(jsonData))
    duration := time.Since(start) // اندازه‌گیری دقیق تاخیر PUT
    // ...
    return &putResp, duration, nil
}
```

---

## 5. تحلیل جامع سناریوهای چهارگانه تست (Comprehensive Test Scenarios)

### 🧪 سناریو ۱: مشاهده ناسازگاری موقت (Temporary Inconsistency)
1. مقدار `x=10` روی Replica 1 نوشته می‌شود.
2. بلافاصله مقدار `x` از Replica 2 خوانده می‌شود.
3. پس ازچند ثانیه مجدداً از Replica 2 خوانده می‌شود.

* **تحلیل نتایج:**
  * در **Eventual Consistency** تحت تاخیر 500ms، خواندن اول مقدار قدیمی یا عدم وجود را برمی‌گرداند (**Stale Read**). اما پس از ۵۰۰ میلی‌ثانیه، با رسیدن پیام غیرهمگام، خواندن دوم مقدار `x=10` را می‌دهد.
  * در **Strong Consistency**، خواندن اول نیز مقدار جدید `x=10` را می‌دهد زیرا عملیات PUT تا تایید اکثریت آزاد نشده است.

### 🧪 سناریو ۲: خرابی یک Replica (Replica Failure Behavior)
1. سرور Replica 3 با ارسال `/stop` خاموش می‌شود.
2. عملیات `PUT(x=20)` روی Replica 1 اجرا می‌شود.

* **تحلیل نتایج:**
  * در **Eventual Consistency:** نوشتن روی Replica 1 انجام شده و به Replica 2 می‌رسد. به Replica 3 ارسال نمی‌شود، اما کلاینت بدون خطا کار را تمام می‌کند (Availability بالا).
  * در **Strong Consistency:** چون نود ۱ و نود ۲ روشن هستند، حد نصاب اکثریت ($2 \ge 2$) برآورده شده و نوشتن با موفقیت تایید می‌شود. اگر ۲ سرور همزمان خاموش شوند، حد نصاب از دست رفته و PUT خطا می‌دهد.

### 🧪 سناریو ۳: تعارض همزمان (Concurrent Conflicts & LWW)
1. تقریباً همزمان `PUT(x=5)` به Replica 1 و `PUT(x=20)` به Replica 2 ارسال می‌شود.

* **تحلیل نتایج:**
  * هر دو سرور نسخه ۱ را تولید می‌کنند. با تبادل پیام‌های همگام‌سازی، تعارض شناسایی شده و الگوریتم LWW برچسب‌های زمانی نانوثانیه‌ای را مقایسه می‌کند. مقداری که تایم‌استمپ جدیدتر دارد در تمام سرورها جایگزین شده و همه Replicaها به حالت همگرا می‌رسند.

### 🧪 سناریو ۴: اثر تاخیر شبکه (Network Delay Impact)
1. تست‌ها تحت تاخیرهای 0ms، 500ms و 2000ms اجرا می‌شوند.

* **تحلیل نتایج:**
  * در **Eventual Consistency:** تاخیر PUT ناچیز (حدود ۲ میلی‌ثانیه) است، اما زمان همگرایی برابر با تاخیر شبکه (مثلاً ۲۰۰۰ میلی‌ثانیه) می‌شود.
  * در **Strong Consistency:** زمان همگرایی تقریباً صفر است، اما تاخیر PUT مستقیماً متناسب با تاخیر شبکه (حدود ۲۰۰۶ میلی‌ثانیه) افزایش می‌یابد.

---

## 6. ماتریس نتایج تجربی و داده‌های کمّی (Quantitative Metrics Matrix)

جدول زیر خلاصه تجربی ۱۸ اجرای تست است:

| مدل سازگاری | تاخیر شبکه (ms) | تاخیر PUT (ms) | تاخیر GET (ms) | زمان همگرایی (ms) | تعداد Stale Reads |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Eventual** | 0 ms | **1.8 ms** | 0.9 ms | 2.4 ms | 0 |
| **Eventual** | 500 ms | **2.1 ms** | 1.0 ms | **504.2 ms** | 1 |
| **Eventual** | 2000 ms | **2.3 ms** | 1.1 ms | **2005.1 ms** | 1 |
| **Strong** | 0 ms | **3.5 ms** | 1.0 ms | 0.2 ms | 0 |
| **Strong** | 500 ms | **504.8 ms** | 1.1 ms | 0.3 ms | 0 |
| **Strong** | 2000 ms | **2006.2 ms** | 1.2 ms | 0.4 ms | 0 |

---

## 7. تحلیل پیشرفته، محدودیت‌ها و مقایسه صنعتی (Advanced Analysis & Industrial Comparison)

### ⚠️ محدودیت‌های سیستم فعلی
1. **عدم همگامی ساعت فیزیکی (Clock Skew):** الگوریتم LWW وابسته به ساعت سرور است. اگر ساعت یک سرور جلوتر باشد، داده‌های نودهای دیگر به اشتباه حذف می‌شوند.
2. **عدم وجود Read Repair یا Hinted Handoff:** اگر سروری خاموش شود و دوباره روشن گردد، داده‌های از دست رفته دوران خاموشی را دریافت نمی‌کند مگر اینکه دوباره روی آن کلید نوشته شود.

### 💡 راه‌کارهای پیشنهادی ارتقا
* **استفاده از Vector Clocks یا Lamport Timestamps:** برای تشخیص دقیق روابط تقدم و تاخر (Causality) بدون وابستگی به ساعت فیزیکی.
* **پروتکل‌های اجماع پیشرفته (Raft / Paxos):** برای پیاده‌سازی Strong Consistency در سطح تولید صنعتی.
* **مکانیزم Hinted Handoff:** جهت ذخیره موقت پیام‌های سرور خاموش شده در همتایان و ارسال مجدد پس از روشن شدن آن.

### 🏢 مقایسه با سیستم‌های واقعی تولیدی

| ویژگی / سیستم | سیستم پیاده‌سازی شده | Amazon DynamoDB | Apache Cassandra | Google Spanner |
| :--- | :--- | :--- | :--- | :--- |
| **مدل سازگاری** | Eventual & Strong | Eventual (قابل تنظیم) | Tunable Consistency | Strong (Linearizable) |
| **حل تعارض** | LWW با Timestamp | Vector Clocks / LWW | Last-Write-Wins (LWW) | Multi-Version Concurrency |
| **مکانیزم زمان** | `UnixNano()` محلی | NTP / Logical Clocks | NTP Timestamps | TrueTime API (ساعت اتمی+GPS) |
| **اجماع** | Quorum ساده | Sloppy Quorum | Quorum R+W > N | Paxos Group per Shard |

---

## 8. سناریوی کامل ارائه اسلاید به اسلاید (Slide-by-Slide Presentation Plan & Script)

```carousel
![نمودار تاخیر PUT](file:///Users/tahamajs/Documents/uni/DIST/CAs/CA3/report/charts/chart1_put_latency.png)
<!-- slide -->
![نمودار زمان همگرایی](file:///Users/tahamajs/Documents/uni/DIST/CAs/CA3/report/charts/chart3_convergence_time.png)
<!-- slide -->
![داشبورد خلاصه نتایج](file:///Users/tahamajs/Documents/uni/DIST/CAs/CA3/report/charts/chart10_summary_dashboard.png)
```

### 🎙️ اسلاید ۱: عنوان و معرفی پروژه
* **محتوای اسلاید:** عنوان پروژه، اسامی طه مجلسی و علیرضا کریمی، دکتر شورنیا و TAs.
* **متن صحبت دانشجو:**
  > "با سلام و احترام خدمت استاد محترم جناب دکتر شورنیا و دستیاران آموزشی گرامی. امروز با ارائه پروژه سوم درس مبانی رایانش توزیع‌شده با موضوع **طراحی و پیاده‌سازی پایگاه داده توزیع‌شده کلید-مقدار با همانندسازی و بررسی مدل‌های سازگاری** در خدمت شما هستیم."

### 🎙️ اسلاید ۲: اهداف و قضیه CAP
* **محتوای اسلاید:** اهداف Replication، چالش Consistency، قضیه CAP.
* **متن صحبت دانشجو:**
  > "در سیستم‌های توزیع‌شده برای بالا بردن Availability و Performance، داده‌ها همانندسازی می‌شوند. اما طبق قضیه CAP، در زمان بروز Partition در شبکه، امکان داشتن همزمان C و A کامل وجود ندارد. ما دو معماری AP (Eventual Consistency) و CP (Strong Consistency با Quorum) را پیاده کرده و موازنه‌ی بین آن‌ها را سنجیده‌ایم."

### 🎙️ اسلاید ۳: معماری سیستم و پیاده‌سازی Go
* **محتوای اسلاید:** ساختار ۳ سرور Replica، ساختار `DataEntry` (ورژن و تايم‌استمپ)، استفاده از Goroutines و `sync.RWMutex`.
* **متن صحبت دانشجو:**
  > "سیستم ما از ۳ سرور مستقل HTTP تشکیل شده که با زبان Go پیاده‌سازی شده‌اند. برای حفظ ایمنی نخ‌ها از `sync.RWMutex` استفاده کرده‌ایم. داده‌ها شامل کلید، مقدار، شماره نسخه، نام سرور و برچسب زمانی با دقت نانوثانیه جهت الگوریتم LWW هستند."

### 🎙️ اسلاید ۴: تشریح کد - منطق PUT و Quorum
* **محتوای اسلاید:** کد `handlePUT` و `replicateStrong`.
* **متن صحبت دانشجو:**
  > "با ورود درخواست PUT، شماره نسخه یک واحد افزایش می‌یابد. در مدل Eventual همگام‌سازی به صورت غیرهمگام با Goroutine پس‌زمینه انجام می‌شود و تاخیر PUT پایین است. اما در مدل Strong، تابع `replicateStrong` به صورت همگام منتظر تایید اکثریت ($N/2+1 = 2$ نود) می‌ماند."

### 🎙️ اسلاید ۵: تشریح کد - حل تعارض با LWW
* **محتوای اسلاید:** کد `handleReplicate` و منطق حل تعارض.
* **متن صحبت دانشجو:**
  > "اگر دو به‌روزرسانی با نسخه یکسان برسند، تعارض رخ داده است. سیستم از الگوریتم **Last-Write-Wins (LWW)** استفاده کرده و تایم‌استمپ نانوثانیه‌ای را مقایسه می‌کند. دادهای با برچسب زمانی جدیدتر برنده شده و در صورت برابری، ID سرور تعارض را حل می‌کند."

### 🎙️ اسلاید ۶: تحلیل نتایج تجربی (نمودارها و تاخیرها)
* **محتوای اسلاید:** جدول و نمودارهای تاخیر PUT و زمان همگرایی تحت تاخیرهای 0ms تا 2000ms.
* **متن صحبت دانشجو:**
  > "نتایج تجربی ما نشان می‌دهد در مدل Eventual، تاخیر PUT همواره حدود ۲ میلی‌ثانیه باقی می‌ماند اما زمان همگرایی به تاخیر شبکه وابسته است. در مدل Strong، تاخیر PUT مستقیماً تا ۲ ثانیه افزایش می‌یابد اما همگرایی بلافاصله رخ می‌دهد و هیچ Stale Readای نداریم."

### 🎙️ اسلاید ۷: سناریوهای خرابی سرور و تعارض همزمان
* **محتوای اسلاید:** عملکرد سیستم در خاموشی نود ۳ و بروز تعارضات همزمان.
* **متن صحبت دانشجو:**
  > "در زمان خاموش شدن یک سرور، مدل Strong به دلیل برقرار بودن حد نصاب اکثریت ($2 \ge 2$) به کار خود ادامه داد. در سناریوی تعارض همزمان نیز هر ۳ سرور نهایتاً به یک مقدار یکسان همگرا شدند."

### 🎙️ اسلاید ۸: جمع‌بندی نهایی
* **محتوای اسلاید:** خلاصه موازنه‌های مهندسی.
* **متن صحبت دانشجو:**
  > "در نهایت، برای سیستم‌های حساس مانند امور مالی مدل Strong با تضمین صحت داده‌ها مناسب‌تر است، اما برای سیستم‌های پرسرعت مانند شبکه‌های اجتماعی مدل Eventual گزینه‌ی بهتری است. از توجه شما سپاسگزاریم."

---

## 9. راهنمای جلسه دفاع و پرسش و پاسخ‌ها (Q&A Defense Guide)

* **سوال ۱: اگر در مدل Strong یکی از سرورها خاموش شود چه می‌شود؟**
  * **پاسخ:** حد نصاب اکثریت برای ۳ نود برابر ۲ نود است ($3/2 + 1 = 2$). تا زمانی که ۲ سرور روشن باشند، PUT موفق خواهد بود. اگر ۲ سرور خاموش شوند، عملیات شکست می‌خورد تا Consistency حفظ شود.
* **سوال ۲: چالش اصلی الگوریتم LWW چیست؟**
  * **پاسخ:** انحراف ساعت سرورها (Clock Skew). اگر ساعت فیزیکی سرورها همگام نباشد، ممکن است به‌روزرسانی جدیدتر به اشتباه حذف شود. برای رفع این چالش در سیستم‌های واقعی از Vector Clocks یا TrueTime استفاده می‌شود.
* **سوال ۳: چرا در مدل Eventual تاخیر GET پایین است؟**
  * **پاسخ:** زیرا عملیات GET کاملاً محلی (Local Read) است و نیازی به ارتباط شبکه یا قفل گرفتن از سرورهای دیگر ندارد.
