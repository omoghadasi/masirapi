# راهنمای نسخه‌گذاری و انتشار فورک

این سند فقط «شماره نسخه» و «انتشار» را پوشش می‌دهد. همگام‌سازی با مخزن رسمی (`upstream`) در [new_api_workflow.md](new_api_workflow.md) توضیح داده شده و آن را دست‌نخورده می‌گذارد.

تصمیم‌های ثابت این سند:

- الگوی نسخه: `vX.Y.Z.N` متصل به نسخهٔ بالادستی (بخش ۳)
- کانال انتشار: `docker build` لوکال؛ بدون GitHub Actions برای ایمیج (بخش ۵)
- همهٔ مراحل با دستور دستی، بدون اسکریپت کمکی

---

## ۱. نسخه از کجا می‌آید (زنجیرهٔ واقعی)

| مرحله | فایل | توضیح |
| --- | --- | --- |
| ۱. مقدار نسخه | [`VERSION`](VERSION) در ریشه | تنها منبع نسخه در build لوکال. **الان صفر بایت است** (کامیت بالادستی `fix: add a blank VERSION file`). |
| ۲. بیک‌کردن در باینری | [Dockerfile:31](Dockerfile) | `-ldflags "-X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'"` |
| ۳. پیش‌فرض | [common/constants.go:14](common/constants.go) | `var Version = "v0.0.0"`؛ اگر لیدفلگ نیاید همان `v0.0.0` می‌ماند. |
| ۴. override زمان اجرا | [common/init.go:35](common/init.go) | متغیر محیطی `VERSION` در صورت غیرخالی بودن بر مقدار build شده غالب می‌شود. |
| ۵. API | [controller/misc.go:51](controller/misc.go) | `GET /api/status` → `data.version = common.Version` |
| ۶. خواندن در فرانت | [web/src/lib/api.ts:73](web/src/lib/api.ts) و [web/src/hooks/use-status.ts](web/src/hooks/use-status.ts) | کش در `localStorage['status']` هم دارد. |
| ۷. نرمال‌سازی | [web/src/features/system-update/use-system-update.ts:151-155](web/src/features/system-update/use-system-update.ts) | `v0.0.0`، `0.0.0` و رشتهٔ خالی → `undefined` |
| ۸. نمایش در هدر | [web/src/components/layout/components/app-header.tsx:118](web/src/components/layout/components/app-header.tsx) + [web/src/features/system-update/system-update-action.tsx:58](web/src/features/system-update/system-update-action.tsx) | `SystemUpdateAction presentation='version'`؛ در نبود نسخه «Unknown version» نشان می‌دهد و **فقط برای ادمین** دیده می‌شود ([`role >= ROLE.ADMIN`](web/src/features/system-update/system-update-action.tsx)). |

مسیر موازی (فقط برای عیب‌یابی، در UI نمایش داده نمی‌شود): [Dockerfile:8](Dockerfile) مقدار `VITE_REACT_APP_VERSION=$(cat VERSION)` را به [web/src/lib/build-metadata.ts:68](web/src/lib/build-metadata.ts) می‌دهد و از آن `data-build-rev`، `<meta name="build-id">` و `localStorage['app:rev']` ساخته می‌شود. اگر `VERSION` خالی باشد، این رشته `rv.0000.2k6e8r7p` می‌شود.

دو نکتهٔ مهم:

- `git describe` داخل ایمیج **کار نمی‌کند**: `.git` در [.dockerignore](.dockerignore) حذف شده است. پس هیچ راهی جز پرکردن فایل `VERSION` قبل از build نیست.
- `docker-compose.yml` شما `build: .` و `image: new-api:local` دارد؛ تگ `local` شناور است و هیچ اطلاعات نسخه‌ای نمی‌دهد. نسخه باید از داخل ایمیج/`/api/status` خوانده شود.

---

## ۲. چرا فقط یک فرمت خاص مجاز است

فرانت نسخه را با این الگو می‌فهمد ([web/src/features/system-update/releases.ts:53](web/src/features/system-update/releases.ts)):

```text
^v?(\d+(?:\.\d+){2,})(?:-(alpha|beta|rc|patch)(?:\.(\d+))?(?:-i18nfix\.(\d+))?)?(?:\+[\da-zA-Z.-]+)?$
```

یعنی:

- حداقل **سه بخش عددی** لازم است: `v1.0.0` ✔، `v1.0` ✘
- پیشوند یا پسوند متنی آزاد مجاز نیست: `masir-v1.0.0.1` ✘، `v1.0.0.1-masir` ✘، `v1.0.0+masir.1` برای پارس ✔ ولی برای Docker tag ✘
- تنها «کلمهٔ» مجاز بعد از `-` یکی از `alpha|beta|rc|patch` است.
- برچسب ایمیج Docker فقط `[A-Za-z0-9_.-]` را می‌پذیرد؛ `+` مجاز نیست.

نتیجه: هر عددی که انتخاب می‌کنی باید از همین قالب باشد، وگرنه هدر «Unknown version» می‌شود و بنر مقایسهٔ نسخه از کار می‌افتد.

---

## ۳. الگوی `vX.Y.Z.N`

```text
v  X . Y . Z . N
   └──┴──┴──┴── upstream baseline که کد روی آن است
            └── شمارهٔ ریلیز فورک روی همان مبنا (از ۱)
```

مثال‌ها:

```text
v1.0.0-rc.47  →  مبنا  : v1.0.0-rc.47 (هنوز stable نشده)
                 ریلیز : v1.0.0.1  اولین بیلد فورک
                 hotfix: v1.0.0.2  اصلاح RTL در کامپوننت‌ها

همگام با upstream v1.1.0  →  v1.1.0.1
Staging قبل از نهایی کردن →  v1.1.0.1-rc.1 سپس v1.1.0.1
```

قواعد:

1. `X.Y.Z` را از نسخهٔ بالادستی مبنا بگیر، نه از سلیقهٔ خودت؛ این کار باعث می‌شود بنر «Update available» درست کار کند: اگر فورک پشت بالادست باشد بنر نمایش داده می‌شود، و وقتی هم‌سطح باشد (`N ≥ 1` نسبت به `vX.Y.Z` خالی) بنر خاموش می‌ماند.
2. `N` فقط با انتشار جدید زیاد می‌شود؛ هرگز `N` موجود را بازنویسی نکن.
3. تگ موجود را جابه‌جا نکن (`git tag -f` و Force Push ممنوع). برای اصلاح، تگ بعدی بزن.
4. بالادست در گذشته گاهی خودش تگ چهارعددی زده (`v0.1.3.1`، `v0.2.0.1`)؛ پس **قبل از هر تگ** تداخل را بررسی کن (مرحله ۵.۲). اگر نامی گرفته شده بود، `N` بعدی را بردار.

وضعیت فعلی مخزن (برای شروع):

- مبنای واقعی `main` شما: `git describe --tags --match 'v[0-9]*' upstream/main` → `v1.0.0-rc.47-38-gea343c4a8`، یعنی `X.Y.Z = 1.0.0`.
- اولین ریلیز این الگو: `v1.0.0.1`.
- تگ `v0.0.1` که روی `origin` داری در این الگو مبنای درستی ندارد (عدد بالادست را نشان نمی‌دهد). پاکش نکن — فقط از این به بعد با `vX.Y.Z.N` تگ بزن و در Release Notes بنویس که خط نسخه عوض شده.

---

## ۴. یک‌بار: آماده‌سازی فورک

1. فایل `VERSION` در اولین ریلیز با مقدار واقعی کامیت می‌شود؛ تا وقتی خالی است، هر `docker build` لوکال نسخهٔ خالی می‌سازد.
2. فایل‌های workflow روی push تگ واکنش نشان می‌دهند و چون فقط build لوکال می‌خواهی، بهتر است آن‌ها را بی‌اثر کنی:
   - [`.github/workflows/docker-build.yml`](.github/workflows/docker-build.yml) — ایمیج را به `calciumion/new-api` (اکانت Docker Hub بالادست) push می‌کند؛ روی فورک یا fail می‌شود یا `denied`. اگر Actions روشن بماند، هر تگ یک اجرای بی‌نتیجه و مصرف quota می‌سازد.
   - [`.github/workflows/electron-build.yml`](.github/workflows/electron-build.yml) — روی هر تگی که `-` نداشته باشد اجرا می‌شود؛ یعنی دقیقاً روی `vX.Y.Z.N`.
   - [`.github/workflows/release.yml`](.github/workflows/release.yml) — باینری‌های Linux/macOS/Windows و GitHub Release می‌سازد؛ نسخه را از خود تگ می‌گیرد و چون فرمت `vX.Y.Z.N` پارس می‌شود، بی‌ضرر و گاهی مفید است.
   - ساده‌ترین حالت: Settings → Actions → General → «Disable actions» در مخزن فورک. حالت دوم: حذف `docker-build.yml` و `electron-build.yml` در یک کامیت فورک و نگه‌داشتن `release.yml`.
3. `sync-release-to-gitcode.yml` و `docker-image-branch.yml` فقط `workflow_dispatch` هستند؛ کاری نکن.
4. هرگز `git push origin --tags` نزن: تگ‌های بالادست (`v1.0.0-rc.*`) داخل مخزن تو هم وجود دارند و با این دستور به `origin` منتقل می‌شوند. همیشه فقط همان یک تگ را push کن.

---

## ۵. مراحل هر ریلیز

### ۵.۱ پیش‌نیاز

```bash
git switch main
git pull --ff-only origin main
git status --porcelain        # باید خالی باشد
```

تست‌های مربوط به آخرین تغییرات باید سبز باشند (`make test` به Go روی سیستم نیاز دارد؛ اگر نداری، همین مراحل را داخل کانتینر `golang` یا در CI اجرا کن):

```bash
make test
cd web && bun run test && bunx tsc --noEmit -p tsconfig.json && cd ..
```

### ۵.۲ تعیین مبنا و بررسی تداخل نام

```bash
# نسخهٔ بالادستی که HEAD فعلی داخل آن است
git describe --tags --match 'v[0-9]*' upstream/main

# نام انتخابی باید در هیچ‌کدام از دو مخزن وجود نداشته باشد
git ls-remote upstream "refs/tags/v1.0.0.1*"
git ls-remote origin   "refs/tags/v1.0.0.1*"
```

خروجی هر دو `ls-remote` باید خالی باشد. اگر خالی نبود، `N` را بالا ببر.

### ۵.۳ نوشتن `VERSION` و کامیت ریلیز

```bash
printf 'v1.0.0.1' > VERSION     # بدون newline و بدون CRLF
cat VERSION; echo
git add VERSION
git commit -m "chore(release): v1.0.0.1"
```

این کامیت «نقطهٔ ریلیز» توست: همان چیزی که داخل ایمیج می‌رود، همان چیزی که در هدر دیده می‌شود.

### ۵.۴ build ایمیج با تگ نسخه

```bash
docker build -t new-api:v1.0.0.1 .
docker tag new-api:v1.0.0.1 new-api:local   # تا compose فعلی بدون تغییر کار کند
```

### ۵.۵ تأیید نسخه، قبل از استقرار

```bash
# چاپ نسخهٔ باینری (پرچم --version در common/init.go:40، قبل از اتصال به دیتابیس)
docker run --rm new-api:v1.0.0.1 --version
```

خروجی باید دقیقاً `v1.0.0.1` باشد. اگر `v0.0.0` یا خط خالی دیدی، یعنی `VERSION` موقع build خالی بوده یا لایهٔ کش‌شدهٔ قدیمی مصرف شده؛ بعد از نوشتن `VERSION` با `docker build --no-cache -t new-api:v1.0.0.1 .` تکرار کن. (نسخهٔ فرانت یعنی `data-build-rev` فقط برای عیب‌یابی است و در هدر دیده نمی‌شود؛ ملاک نهایی همان `--version` و `/api/status` است.)

### ۵.۶ تگ Git و push

```bash
git tag -a v1.0.0.1 -m "MasirAPI v1.0.0.1 (upstream base v1.0.0-rc.47)"
git push origin main
git push origin v1.0.0.1
```

### ۵.۷ استقرار

قبل از هر چیز Backup ([new_api_workflow.md](new_api_workflow.md))، بعد:

```bash
docker compose up -d new-api
curl -s localhost:3000/api/status | grep -o '"version":"[^"]*"'
```

خروجی `curl` باید `"version":"v1.0.0.1"` باشد. سپس با اکانت ادمین صفحه را باز کن: pill کنار لوگوی هدر باید `v1.0.0.1` را نشان دهد.

در محیط پروداکشن، `image` را روی تگ نسخه پین کن و `build: .` را از سرویس حذف کن تا اشتباهاً با کد working tree build نشود:

```yaml
  new-api:
    image: new-api:v1.0.0.1
```

### ۵.۸ Rollback

```bash
docker compose up -d --no-build new-api   # با image: new-api:v1.0.0.1 قبلی در compose
```

چون هر ریلیز یک ایمیج تگ‌دار مستقل است، Rollback فقط برگرداندن همان `image:` به تگ قبلی است (به‌علاوهٔ بازگرداندن Backup اگر migration خورده شده).

---

## ۶. عیب‌یابی

| نشانه | علت و بررسی |
| --- | --- |
| هدر «Unknown version» | `VERSION` موقع build خالی بوده: `docker run --rm IMG --version`. اصلاح: مرحلهٔ ۵.۳ تا ۵.۵ را دوباره اجرا کن. |
| هدر نسخهٔ قبلی را نشان می‌دهد | کش `localStorage['status']` در فرانت، یا ایمیج قدیمی در حال اجرا. Hard refresh و `docker compose up -d` با تگ جدید. |
| `data-build-rev` برابر `rv.0000.…` | `VITE_REACT_APP_VERSION` خالی بوده → `VERSION` در build نبوده؛ همان ریشهٔ مورد بالا. |
| بنر «Update available: vX.Y.Z» همیشه | این مقایسه با ریلیزهای `QuantumNous/new-api` انجام می‌شود و عملاً یادآوری همگام‌سازی است. اگر آزاردهنده است، آن نسخه را «ignore» کن؛ برای حذفش عدد نسخه را زورکی بالا نبر — باعث می‌شود شماره از بالادست جلو بزند و معنای مبنا را از دست بدهد. |
| `parse error` در build یا نسخه با `\r` چسبیده | فایل `VERSION` را CRLF نساز؛ `.gitattributes` با `text=auto` است. با `printf` بنویس و `cat -A VERSION` را ببین. |
| build در مرحلهٔ `go mod download` می‌ماند | `GOPROXY=https://goproxy.cn,...` در [Dockerfile](Dockerfile) هاردکد شده (نه `ARG`) و شبکهٔ ایمیج ممکن است به آن برسد ولی به `proxy.golang.org` نرسد. با `docker build --network=host` امتحان کن؛ اگر لازم بود همان خط `ENV GOPROXY` را به mirror خودت تغییر بده و در یک کامیت جدا نگه‌دار تا در sync بعدی گم نشود. |
| تگ غلط زده شد و push هم شد | تگ را جابه‌جا نکن. `vX.Y.Z.(N+1)` بساز و در Release Notes توضیح بده. |
| ایمیجی داری که نسخه داخلش خالی است و نمی‌خواهی rebuild کنی | متغیر محیطی `VERSION` در [common/init.go:35](common/init.go) بر مقدار build‌شده غالب می‌شود؛ به سرویس در compose اضافه کن و دوباره بالا بیاور: `- VERSION=v1.0.0.1`. هدر درست می‌شود، ولی `data-build-rev` فرانت همچنان `rv.0000.…` می‌ماند؛ پس این فقط راه‌حل موقت است و ریلیز بعدی را با `VERSION` درست build کن. |

---

## ۷. چک‌لیست هر ریلیز

- [ ] `main` تازه است و Working Tree تمیز.
- [ ] تست‌های backend (`make test`) و frontend (vitest + tsc) سبز.
- [ ] `git describe --tags --match 'v[0-9]*' upstream/main` → مبنای نسخه مشخص شد.
- [ ] `git ls-remote upstream/origin "refs/tags/vX.Y.Z.N*"` خالی بود.
- [ ] `VERSION` با `printf` نوشته و در کامیت ریلیز ذخیره شد.
- [ ] `docker build -t new-api:vX.Y.Z.N .` موفق.
- [ ] `docker run --rm new-api:vX.Y.Z.N --version` همان نسخه را چاپ کرد.
- [ ] `curl /api/status` همان `version` را برگرداند.
- [ ] هدر (با اکانت ادمین) نسخه را نشان می‌دهد.
- [ ] تگ annotated ساخته و **فقط همان تگ** push شد.
- [ ] Backup دیتابیس گرفته شد و `image:` پروداکشن روی تگ جدید پین شد.
- [ ] امکان Rollback به تگ قبلی تست/ثبت شد.
