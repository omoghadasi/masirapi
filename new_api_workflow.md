# راهنمای نگهداری و به‌روزرسانی Fork پروژه New API

این سند راهنمای عملیاتی تیم برای نگهداری Fork اختصاصی از [New API](https://github.com/QuantumNous/new-api) است. هدف این است که تغییرات نسخه رسمی را دریافت و بررسی کنیم، آن‌ها را با توسعه‌های اختصاصی مقایسه کنیم، تعارض‌ها را حل کنیم و فقط پس از تست به نسخه خودمان منتقل کنیم.

> **قاعده اصلی:** تغییرات رسمی را ابتدا در یک شاخه موقت بررسی کنید؛ مستقیم روی `main` یا شاخه Production ادغام نکنید. قبل از شروع، از تغییرات محلی و دیتابیس محیط عملیاتی نسخه پشتیبان بگیرید.

---

## ۱. واژه‌ها و ساختار شاخه‌ها

- `origin`: مخزن اختصاصی تیم شما؛ مقصد Push.
- `upstream`: مخزن رسمی `QuantumNous/new-api`؛ منبع دریافت تغییرات.
- `upstream/main`: آخرین وضعیت شاخه `main` در مخزن رسمی، پس از `fetch`.
- `main`: نسخه پایدار و قابل انتشار پروژه شما.
- `develop`: شاخه یکپارچه‌سازی و تست تغییرات تیم.
- `upstream-sync/*`: شاخه موقت برای بررسی و ادغام تغییرات رسمی.
- `feature/*` و `fix/*`: شاخه‌های قابلیت جدید و رفع اشکال.

جریان پیشنهادی:

```text
QuantumNous/new-api (upstream/main)
             |
          git fetch
             |
             v
upstream-sync/YYYY-MM-DD  -- بررسی و مقایسه
             |
       حل تعارض + تست
             |
             v
          develop  -- تست Staging
             |
        Pull Request
             |
             v
           main  -- Release Tag
             |
             v
        Docker / Production
```

> نام شاخه رسمی را فرض نکنید. اگر مخزن رسمی به‌جای `main` از شاخه دیگری استفاده می‌کند، دستورات را با نام واقعی آن شاخه جایگزین کنید.

---

## ۲. فقط یک بار: آماده‌سازی مخزن

### ۲.۱. دریافت مخزن اختصاصی

اگر هنوز مخزن را Clone نکرده‌اید:

```bash
git clone https://github.com/YOUR-ORG/new-api.git
cd new-api
```

آدرس بالا را با آدرس واقعی Fork خودتان عوض کنید.

اگر پروژه را قبلاً Clone کرده‌اید، وارد پوشه همان مخزن شوید:

```bash
cd /path/to/new-api
```

### ۲.۲. بررسی Remoteها

```bash
git remote -v
git status
git branch -vv
```

باید `origin` به مخزن خودتان اشاره کند. اگر `upstream` از قبل تعریف نشده، اضافه‌اش کنید:

```bash
git remote add upstream https://github.com/QuantumNous/new-api.git
```

اگر خطای `remote upstream already exists` دیدید، دوباره اضافه نکنید؛ آدرس موجود را بررسی کنید:

```bash
git remote get-url upstream
```

در صورت نیاز آدرس را اصلاح کنید:

```bash
git remote set-url upstream https://github.com/QuantumNous/new-api.git
```

در پایان:

```bash
git remote -v
```

نمونه خروجی مورد انتظار:

```text
origin    https://github.com/YOUR-ORG/new-api.git (fetch)
origin    https://github.com/YOUR-ORG/new-api.git (push)
upstream  https://github.com/QuantumNous/new-api.git (fetch)
upstream  https://github.com/QuantumNous/new-api.git (push)
```

### ۲.۳. بررسی شاخه رسمی و وضعیت محلی

```bash
git fetch origin --prune
git fetch upstream --prune --tags
git branch -a
git log --oneline --decorate -12
git status --short
```

در ادامه فرض می‌کنیم شاخه رسمی `upstream/main` و شاخه پایدار شما `main` است.

اگر `git status` تغییرات محلی نشان می‌دهد، پیش از ادامه آن‌ها را Commit یا با روش امنی ذخیره کنید. روی تغییرات ناشناخته `reset --hard` اجرا نکنید.

### ۲.۴. ایجاد شاخه‌های تیم (در صورت نیاز)

اگر `main` و `develop` در مخزن شما وجود دارند، این مرحله را رد کنید. اگر `develop` ندارید و `main` معتبر است:

```bash
git switch main
git pull --ff-only origin main
git switch -c develop
git push -u origin develop
```

این دستورات فرض می‌کنند `origin/main` وجود دارد و با تاریخچه مخزن شما سازگار است. اگر هنوز Fork را ایجاد نکرده‌اید یا شاخه‌ها متفاوت‌اند، ابتدا وضعیت واقعی مخزن را بررسی کنید.

---

## ۳. هر بار: دریافت نسخه جدید رسمی

این مرحله فقط اطلاعات Remote را به‌روزرسانی می‌کند؛ هنوز فایل‌های شاخه کاری شما تغییر نمی‌کنند.

### ۳.۱. مطمئن شوید محیط کاری تمیز است

```bash
git status
```

اگر تغییرات Commitنشده دارید، آن‌ها را Commit کنید یا موقتاً با `git stash push -u -m "before upstream sync"` ذخیره کنید. قبل از `stash pop` وضعیت را بررسی کنید.

### ۳.۲. دریافت اطلاعات تازه از مخازن

```bash
git fetch upstream --prune --tags
git fetch origin --prune
```

بررسی کنید آخرین Commit رسمی چیست:

```bash
git log -1 --oneline upstream/main
git log -1 --oneline origin/main
git log -1 --oneline main
```

برای دیدن Commitهای رسمی که در `main` خودتان نیستند:

```bash
git log --oneline --decorate main..upstream/main
```

برای دیدن اختلاف کلی تعداد Commitها:

```bash
git rev-list --left-right --count main...upstream/main
```

عدد سمت چپ تعداد Commitهای مختص `main` شما و عدد سمت راست تعداد Commitهای مختص `upstream/main` است؛ این شمارش به‌تنهایی نشان نمی‌دهد که کدام تغییر فایل مهم‌تر است.

---

## ۴. ساخت شاخه موقت برای همگام‌سازی

هر بار نام جدیدی بسازید تا بررسی قابل ردیابی باشد. مثال:

```bash
git switch main
git switch -c upstream-sync/2026-10-28
```

تاریخ را با روز واقعی اجرا جایگزین کنید. اگر می‌خواهید از آخرین نسخه پایدار محلی شروع کنید، مطمئن شوید `main` به‌روز و Commitشده است. اگر تیم شما فرایند همگام‌سازی را از `develop` شروع می‌کند، این تصمیم را ثابت و مستند نگه دارید؛ در این راهنما مبنا `main` است.

> اگر `upstream-sync/2026-10-28` از قبل وجود دارد، نام دیگری انتخاب کنید یا پس از اطمینان از اینکه شاخه قدیمی دیگر لازم نیست، آن را مدیریت کنید.

---

## ۵. مقایسه نسخه رسمی با پروژه خودمان

قبل از ادغام، تغییرات را در چند سطح بررسی کنید.

### ۵.۱. فهرست Commitهای جدید رسمی

```bash
git log --oneline --decorate HEAD..upstream/main
```

جزئیات یک Commit:

```bash
git show --stat COMMIT_SHA
git show COMMIT_SHA
```

`COMMIT_SHA` را با شناسه واقعی Commit جایگزین کنید.

### ۵.۲. مقایسه فایل‌ها و خطوط تغییرکرده

برای دیدن خلاصه تغییرات:

```bash
git diff --stat HEAD..upstream/main
git diff --name-status HEAD..upstream/main
```

برای مشاهده تغییرات کامل:

```bash
git diff HEAD..upstream/main
```

برای دیدن تغییرات یک فایل مشخص:

```bash
git diff HEAD..upstream/main -- path/to/file
```

مثال:

```bash
git diff HEAD..upstream/main -- web/
git diff HEAD..upstream/main -- relay/
git diff HEAD..upstream/main -- go.mod
```

این فرمان‌ها تفاوت شاخه فعلی با شاخه رسمی را نشان می‌دهند؛ بنابراین تغییرات اختصاصی شما نیز ممکن است در خروجی دیده شوند. برای بررسی دقیق‌تر، از `git log` و مقایسه سه‌طرفه بخش بعدی استفاده کنید.

### ۵.۳. مقایسه سه‌طرفه: نسخه مشترک، نسخه شما، نسخه رسمی

این فرمان نقطه مشترک تاریخچه دو شاخه را پیدا کرده و اختلاف هر دو طرف را نشان می‌دهد:

```bash
git merge-base HEAD upstream/main
git diff --stat $(git merge-base HEAD upstream/main) HEAD
git diff --stat $(git merge-base HEAD upstream/main) upstream/main
```

- فرمان اول: Commit مشترک پایه.
- فرمان دوم: تغییرات اختصاصی شما از نقطه مشترک تا شاخه فعلی.
- فرمان سوم: تغییرات رسمی از نقطه مشترک تا نسخه رسمی فعلی.

برای مشاهده جزئیات هر طرف:

```bash
BASE=$(git merge-base HEAD upstream/main)
git diff "$BASE" HEAD
git diff "$BASE" upstream/main
```

این روش برای فهمیدن اینکه چه چیزی را تیم شما تغییر داده و پروژه اصلی چه چیزی را تغییر داده، بسیار مفید است.

### ۵.۴. فایل‌های حساس را جداگانه بررسی کنید

پیش از ادغام، به‌ویژه این موارد را بررسی کنید:

- `go.mod` و `go.sum`
- فایل‌های قفل و وابستگی Frontend مانند `package.json` و lockfile موجود در پروژه
- `Dockerfile` و فایل‌های Compose
- `model/` و تغییرات دیتابیس یا Migrationها
- `router/`، `middleware/` و کنترل دسترسی
- `relay/` و تغییرات مرتبط با Providerها
- `web/` و تغییرات رابط کاربری
- فایل‌های تنظیمات، متغیرهای محیطی و روش اجرای سرویس

مسیرها مثال‌اند؛ ساختار دقیق را با نسخه فعلی مخزن تطبیق دهید.

### ۵.۵. قبل از ادغام، تصمیم بگیرید

برای هر تغییر رسمی یکی از این تصمیم‌ها را ثبت کنید:

- پذیرفته می‌شود و بدون تغییر ادغام می‌شود.
- پذیرفته می‌شود ولی نیازمند حل تعارض یا بازنویسی تغییر اختصاصی است.
- نیازمند بررسی بیشتر یا تست اضافه است.
- فعلاً قابل ادغام نیست و باید علت آن در Pull Request مستند شود.

بهتر است خلاصه بررسی را در توضیحات Pull Request بنویسید.

---

## ۶. ادغام تغییرات رسمی در شاخه موقت

پس از بررسی، تغییرات رسمی را در شاخه `upstream-sync/*` ادغام کنید:

```bash
git merge --no-commit --no-ff upstream/main
```

این فرمان ادغام را شروع می‌کند ولی قبل از ساخت Commit نهایی مکث می‌کند. این کار فرصت بررسی تغییرات و تعارض‌ها را می‌دهد.

اگر همه چیز درست بود و تعارضی وجود نداشت، فایل‌های تغییرکرده را بررسی کنید:

```bash
git status
git diff --cached --stat
git diff --cached
```

سپس Commit بسازید:

```bash
git commit -m "chore: merge upstream New API updates"
```

اگر تعارض وجود داشت، مراحل بخش بعد را انجام دهید.

> اگر می‌خواهید ادغام را متوقف کنید، تا زمانی که Merge در حال انجام است می‌توانید از `git merge --abort` استفاده کنید. قبل از شروع، تغییرات محلی را Commit یا ذخیره کنید.

---

## ۷. حل تعارض‌ها (Merge Conflicts)

ابتدا فهرست فایل‌های متعارض را ببینید:

```bash
git status
git diff --name-only --diff-filter=U
```

داخل فایل متعارض ممکن است نشانه‌هایی شبیه این ببینید:

```text
<<<<<<< HEAD
کد فعلی شاخه اختصاصی شما
=======
کد نسخه رسمی
>>>>>>> upstream/main
```

این نشانه‌ها باید با تصمیم آگاهانه حذف شوند. کد دو طرف را کورکورانه انتخاب نکنید؛ ممکن است لازم باشد هر دو تغییر با هم ترکیب شوند.

### ۷.۱. مقایسه نسخه‌های متعارض

در حین Merge، Git این نسخه‌ها را در Index نگه می‌دارد:

- Stage 1: نسخه پایه مشترک
- Stage 2: نسخه فعلی شاخه شما (`ours`)
- Stage 3: نسخه‌ای که ادغام می‌کنید (`theirs`)

برای دیدن هر نسخه از یک فایل:

```bash
git show :1:path/to/file
git show :2:path/to/file
git show :3:path/to/file
```

مسیر را با فایل متعارض جایگزین کنید. اگر Stage 1 وجود نداشته باشد، ممکن است فایل جدید یا حذف‌شده باشد.

### ۷.۲. پس از اصلاح فایل

```bash
git add path/to/resolved-file
git status
```

این کار را برای تمام فایل‌های متعارض تکرار کنید. سپس مطمئن شوید هیچ تعارض حل‌نشده‌ای نمانده:

```bash
git diff --name-only --diff-filter=U
```

اگر خروجی خالی بود، تعارض‌های ثبت‌شده حل شده‌اند. این به‌تنهایی صحت کد را تضمین نمی‌کند؛ تست لازم است.

در پایان:

```bash
git diff --cached
git commit -m "chore: resolve conflicts while syncing upstream"
```

اگر Merge را با `--no-commit` آغاز کرده‌اید، پس از حل تعارض‌ها می‌توانید Commit نهایی ادغام را بسازید. اگر به هر دلیل می‌خواهید کل ادغام را لغو کنید:

```bash
git merge --abort
```

---

## ۸. تست قبل از ادغام به شاخه تیم

روش دقیق تست را با نسخه فعلی New API و مستندات همان نسخه هماهنگ کنید. حداقل این موارد را بررسی کنید:

1. Backend با نسخه Go موردنیاز پروژه Build شود.
2. تست‌های موجود Go اجرا شوند.
3. وابستگی‌های Frontend نصب و Build شوند.
4. Lint یا Type Checkهای تعریف‌شده در پروژه اجرا شوند.
5. Docker Image ساخته شود.
6. سرویس در محیط Staging بالا بیاید.
7. ورود و احراز هویت، پنل کاربر و ادمین، درخواست‌های مدل، لاگ‌ها و مسیرهای اصلی بررسی شوند.
8. تغییرات دیتابیس و Migrationها روی نسخه آزمایشی بررسی شوند.
9. قابلیت‌های اختصاصی شما، از جمله هر تغییری در قیمت‌گذاری، Providerها و پرداخت، دوباره تست شوند.

برای یافتن فرمان‌های رسمی پروژه، فایل‌های `README.md`، `CONTRIBUTING.md`، `Makefile`، `go.mod` و `package.json` را بررسی کنید. فرمان تست یا Build را بدون بررسی ساختار نسخه فعلی فرض نکنید.

---

## ۹. ارسال شاخه همگام‌سازی و Pull Request

بعد از حل تعارض‌ها و تست اولیه:

```bash
git push -u origin upstream-sync/2026-10-28
```

در GitHub یا GitLab یک Pull Request/Merge Request باز کنید:

- **Source:** `upstream-sync/2026-10-28`
- **Target:** `develop`

در توضیحات PR ثبت کنید:

- نسخه/Commit رسمی مبنا
- خلاصه تغییرات رسمی
- فایل‌های متعارض و روش حل آن‌ها
- تست‌هایی که اجرا شده‌اند
- مواردی که هنوز نیاز به بررسی دارند

پس از Code Review و تأیید تست‌ها، PR را Merge کنید. اگر سیاست تیم این است که تغییرات ابتدا مستقیماً روی `main` آماده شوند، این سیاست را از ابتدا انتخاب و یکدست اجرا کنید؛ از ادغام‌های تصادفی بین شاخه‌ها اجتناب کنید.

پس از ادغام در `develop`، Staging را Deploy و تست کنید. تنها پس از تأیید، یک PR جدا از `develop` به `main` باز کنید.

---

## ۱۰. انتشار نسخه اختصاصی

قبل از Release:

```bash
git switch main
git pull --ff-only origin main
git status
```

مطمئن شوید Working Tree تمیز است و Commit موردنظر در `main` قرار دارد. سپس یک Tag جدید بسازید؛ نام Tag باید با سیاست نسخه‌گذاری تیم هماهنگ باشد:

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

اگر Tag وجود دارد، آن را بازنویسی نکنید؛ برای انتشار اصلاحی Tag جدید بسازید. Docker Image را از همین Tag یا Commit مشخص Build کنید و در محیط Production نیز همان نسخه مشخص را اجرا کنید، نه یک Tag شناور مانند `latest`.

پیش از Deploy عملیاتی:

- از دیتابیس و فایل‌های موردنیاز Backup بگیرید.
- Release Notes را ثبت کنید.
- امکان Rollback به Image قبلی را آماده کنید.
- پس از Deploy، Health Check و لاگ‌ها را بررسی کنید.

---

## ۱۱. اگر Fork شما از تاریخچه رسمی جدا شده باشد

اگر `git merge upstream/main` خطا داد، تاریخچه‌ها نامرتبط به نظر رسیدند یا تعداد تعارض‌ها غیرعادی بود، فوراً از `--allow-unrelated-histories` استفاده نکنید.

ابتدا بررسی کنید:

```bash
git log --graph --oneline --decorate --all -30
git merge-base main upstream/main
git remote -v
```

اگر `git merge-base` هیچ خروجی نداد، دو شاخه ممکن است Commit مشترک نداشته باشند. در این شرایط:

1. از مخزن و شاخه‌های فعلی نسخه پشتیبان بگیرید.
2. مشخص کنید مخزن شما واقعاً Fork رسمی است یا از یک ZIP/کپی مستقل شروع شده.
3. تاریخچه و شاخه پیش‌فرض مخزن رسمی را بررسی کنید.
4. اگر لازم است، یک Clone تازه از Fork رسمی در پوشه جدا بسازید.
5. تفاوت کد فعلی خودتان را با تاریخچه رسمی بررسی کنید و سپس مسیر مهاجرت را طراحی کنید.

**از `git reset --hard`، حذف شاخه‌ها یا Force Push روی مخزن اصلی بدون درک کامل اثر آن‌ها استفاده نکنید.**

---

## ۱۲. چک‌لیست کوتاه هر بار به‌روزرسانی

این بخش را می‌توانید در هر نوبت تیک بزنید:

- [ ] Working Tree تمیز است یا تغییرات ذخیره شده‌اند.
- [ ] Backup لازم تهیه شده است.
- [ ] `git fetch upstream --prune --tags` اجرا شده است.
- [ ] Commitهای جدید رسمی با `git log` بررسی شده‌اند.
- [ ] تفاوت فایل‌ها با `git diff` بررسی شده است.
- [ ] شاخه `upstream-sync/YYYY-MM-DD` ساخته شده است.
- [ ] تغییرات رسمی در شاخه موقت ادغام شده‌اند.
- [ ] تمام تعارض‌ها بررسی و حل شده‌اند.
- [ ] Backend، Frontend و Docker Build تست شده‌اند.
- [ ] محیط Staging و قابلیت‌های اختصاصی تست شده‌اند.
- [ ] Pull Request به `develop` بازبینی و ادغام شده است.
- [ ] پس از تست Staging، PR به `main` ادغام شده است.
- [ ] Tag نسخه و Docker Image مشخص ساخته شده‌اند.
- [ ] Backup، Health Check و برنامه Rollback برای Production آماده‌اند.

---

## ۱۳. الگوی نام‌گذاری و ثبت تغییرات

نمونه نام شاخه‌ها:

```text
upstream-sync/2026-10-28
feature/persian-ui
feature/persian-payment
feature/provider-routing
fix/user-base-url
```

نمونه عنوان Pull Request:

```text
chore: sync upstream New API updates (2026-10-28)
```

نمونه قالب توضیحات Pull Request:

```markdown
## خلاصه
- نسخه رسمی مبنا:
- Commit رسمی:
- خلاصه تغییرات:

## تعارض‌ها
- فایل:
- تصمیم و دلیل:

## تست‌ها
- [ ] Backend
- [ ] Frontend
- [ ] Docker Build
- [ ] Staging
- [ ] قابلیت‌های اختصاصی

## ریسک‌ها و نکات انتشار
- Migration دیتابیس:
- تغییر تنظیمات:
- روش Rollback:
```

## ۱۴. نکته نهایی

این فرایند برای **Merge کردن** تغییرات رسمی به Fork طراحی شده است، نه برای Rebase کردن مداوم کل تاریخچه اختصاصی. Merge معمولاً تاریخچه را حفظ می‌کند و برای تیمی که می‌خواهد تغییرات رسمی را به‌صورت دوره‌ای دریافت کند، قابل‌ردیابی‌تر است.

هر بار که نسخه رسمی جدید می‌گیرید، اول آن را مقایسه کنید، بعد ادغام کنید، سپس تست کنید و در نهایت منتشر کنید. هیچ‌وقت صرفاً به‌دلیل موفق بودن Git Merge فرض نکنید که برنامه از نظر عملکردی سالم است.
