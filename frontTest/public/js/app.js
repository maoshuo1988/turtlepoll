const STORAGE_BACKEND = 'fronttest_backend_base_url';
const STORAGE_TOKEN = 'fronttest_user_token';
const STORAGE_ROTATE_CAPTCHA_ENABLED = 'fronttest_rotate_captcha_enabled';
const STORAGE_IMAGE_CAPTCHA_ENABLED = 'fronttest_image_captcha_enabled';
const DEFAULT_BACKEND_BASE_URL = 'http://127.0.0.1:8082';

function getRotateCaptchaEnabled() {
  const raw = localStorage.getItem(STORAGE_ROTATE_CAPTCHA_ENABLED);
  if (raw === null || raw === undefined || raw === '') {
    return true;
  }
  return raw === 'true' || raw === '1' || raw === 'yes';
}

function setRotateCaptchaEnabled(enabled) {
  console.log(111111);
  localStorage.setItem(STORAGE_ROTATE_CAPTCHA_ENABLED, enabled ? 'true' : 'false');
}

function getImageCaptchaEnabled() {
  const raw = localStorage.getItem(STORAGE_IMAGE_CAPTCHA_ENABLED);
  if (raw === null || raw === undefined || raw === '') {
    return true;
  }
  return raw === 'true' || raw === '1' || raw === 'yes';
}

function setImageCaptchaEnabled(enabled) {
  localStorage.setItem(STORAGE_IMAGE_CAPTCHA_ENABLED, enabled ? 'true' : 'false');
}

function normalizeBaseUrl(url) {
  const trimmed = String(url || '').trim();
  if (!trimmed) {
    return '';
  }
  // 允许用户只填 host:port 或 IP:port
  const withScheme = /^https?:\/\//i.test(trimmed) ? trimmed : `http://${trimmed}`;

  // 去掉尾部斜杠，避免 buildUrl 拼接出现 //api
  const normalized = withScheme.replace(/\/+$/, '');

  // 基础校验：必须是 http(s) 且包含 host
  try {
    const u = new URL(normalized);
    if (!/^https?:$/.test(u.protocol) || !u.host) {
      return '';
    }
  } catch (e) {
    return '';
  }
  return normalized;
}

function getBackendBaseUrl() {
  const stored = normalizeBaseUrl(localStorage.getItem(STORAGE_BACKEND) || '');
  if (!stored) {
    return DEFAULT_BACKEND_BASE_URL;
  }
  return stored;
}

function setBackendBaseUrl(url) {
  const normalized = normalizeBaseUrl(url);
  if (!normalized) {
    // 输入无效则清空，让 getBackendBaseUrl 回退到默认值
    localStorage.removeItem(STORAGE_BACKEND);
    return DEFAULT_BACKEND_BASE_URL;
  }
  localStorage.setItem(STORAGE_BACKEND, normalized);
  return normalized;
}

function saveToken(token) {
  if (token) {
    localStorage.setItem(STORAGE_TOKEN, token);
  }
}

function getToken() {
  return localStorage.getItem(STORAGE_TOKEN) || '';
}

function buildUrl(path) {
  console.log(`${getBackendBaseUrl()}${path}`);
  return `${getBackendBaseUrl()}${path}`;
}

async function requestJson(url, options) {
  const finalOptions = {
    credentials: 'include',
    ...options
  };

  let res;
  try {
    res = await fetch(url, finalOptions);
  } catch (err) {
    throw new Error(`请求失败(${url}): ${err.message || err}`);
  }
  const text = await res.text();
  let payload;
  try {
    payload = JSON.parse(text);
  } catch (e) {
    throw new Error(`接口未返回 JSON: ${text.slice(0, 180)}`);
  }

  if (!res.ok) {
    const msg = payload.message || payload.msg || `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return payload;
}

function unwrapApi(payload) {
  if (payload == null) {
    throw new Error('空响应');
  }
  if (typeof payload.success === 'boolean') {
    if (!payload.success) {
      throw new Error(payload.message || payload.msg || '接口返回失败');
    }
    return payload.data !== undefined ? payload.data : payload;
  }
  if (typeof payload.code === 'number') {
    if (payload.code !== 0) {
      throw new Error(payload.message || payload.msg || `接口错误 code=${payload.code}`);
    }
    return payload.data !== undefined ? payload.data : payload;
  }
  return payload.data !== undefined ? payload.data : payload;
}

function normalizeImageSrc(rawValue) {
  const value = String(rawValue || '').trim();
  if (!value) {
    return '';
  }
  if (/^data:image\/[a-zA-Z0-9.+-]+;base64,/i.test(value)) {
    return value;
  }
  return `data:image/png;base64,${value}`;
}

async function getRotateCaptcha() {
  const payload = await requestJson(buildUrl('/api/captcha/request_angle'));
  const data = unwrapApi(payload);
  if (!data || !data.id || !data.imageBase64 || !data.thumbBase64) {
    throw new Error('旋转验证码接口返回缺少字段');
  }
  return data;
}

async function getImageCaptcha() {
  const payload = await requestJson(buildUrl('/api/captcha/request_image'));
  const data = unwrapApi(payload);
  // request_image 是 Put("captchaId"/"captchaBase64") 的返回结构
  if (!data || !data.captchaId || !data.captchaBase64) {
    throw new Error('图片验证码接口返回缺少字段');
  }
  return data;
}

function openImageCaptcha() {
  return new Promise(async (resolve, reject) => {
    let captchaData;
    try {
      captchaData = await getImageCaptcha();
    } catch (err) {
      reject(err);
      return;
    }

    const overlay = document.createElement('div');
    overlay.className = 'ft-captcha-overlay';
    overlay.innerHTML = `
      <div class="ft-captcha-modal">
        <div class="ft-captcha-header">
          <strong>请输入图片验证码</strong>
          <button type="button" class="ft-captcha-close">×</button>
        </div>
        <div class="ft-captcha-body">
          <div style="display:flex;gap:12px;align-items:center;flex-wrap:wrap;">
            <img class="ft-image-captcha" alt="captcha" style="height: 44px; border-radius: 6px; border: 1px solid #e6e6e6;" />
            <button type="button" class="ft-captcha-refresh">刷新</button>
          </div>
          <div style="margin-top:12px;">
            <input class="ft-image-captcha-input" type="text" placeholder="输入验证码" style="width:100%;padding:10px 12px;box-sizing:border-box;" />
          </div>
          <div class="ft-captcha-tip">看不清？点“刷新”换一张</div>
        </div>
        <div class="ft-captcha-actions">
          <button type="button" class="ft-captcha-cancel">取消</button>
          <button type="button" class="ft-captcha-confirm">确认</button>
        </div>
      </div>
    `;
    document.body.appendChild(overlay);

    const imageEl = overlay.querySelector('.ft-image-captcha');
    const inputEl = overlay.querySelector('.ft-image-captcha-input');
    const refreshBtn = overlay.querySelector('.ft-captcha-refresh');
    const cancelBtn = overlay.querySelector('.ft-captcha-cancel');
    const closeBtn = overlay.querySelector('.ft-captcha-close');
    const confirmBtn = overlay.querySelector('.ft-captcha-confirm');

    let finished = false;

    function cleanup() {
      if (overlay.parentNode) {
        overlay.parentNode.removeChild(overlay);
      }
    }

    function rejectAndClose(err) {
      if (finished) return;
      finished = true;
      cleanup();
      reject(err);
    }

    function resolveAndClose(result) {
      if (finished) return;
      finished = true;
      cleanup();
      resolve(result);
    }

    function loadCaptcha(data) {
      captchaData = data;
      imageEl.src = normalizeImageSrc(data.captchaBase64);
      inputEl.value = '';
      setTimeout(() => inputEl.focus(), 0);
    }

    refreshBtn.addEventListener('click', async () => {
      try {
        const data = await getImageCaptcha();
        loadCaptcha(data);
      } catch (err) {
        rejectAndClose(err);
      }
    });
    cancelBtn.addEventListener('click', () => rejectAndClose(new Error('用户取消验证码')));
    closeBtn.addEventListener('click', () => rejectAndClose(new Error('用户取消验证码')));
    overlay.addEventListener('click', (ev) => {
      if (ev.target === overlay) {
        rejectAndClose(new Error('用户取消验证码'));
      }
    });

    confirmBtn.addEventListener('click', () => {
      const code = String(inputEl.value || '').trim();
      if (!code) {
        rejectAndClose(new Error('请输入验证码'));
        return;
      }
      resolveAndClose({
        captchaId: captchaData.captchaId,
        captchaCode: code,
        captchaProtocol: 3
      });
    });

    inputEl.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter') {
        confirmBtn.click();
      }
    });

    loadCaptcha(captchaData);
  });
}

function openRotateCaptcha() {
  return new Promise(async (resolve, reject) => {
    let captchaData;
    try {
      captchaData = await getRotateCaptcha();
    } catch (err) {
      reject(err);
      return;
    }

    const overlay = document.createElement('div');
    overlay.className = 'ft-captcha-overlay';

    overlay.innerHTML = `
      <div class="ft-captcha-modal">
        <div class="ft-captcha-header">
          <strong>请拖拽滑块完成验证</strong>
          <button type="button" class="ft-captcha-close">×</button>
        </div>
        <div class="ft-captcha-body">
          <div class="ft-captcha-image-wrap">
            <img class="ft-captcha-image" alt="captcha-image" />
            <div class="ft-captcha-thumb-wrap">
              <img class="ft-captcha-thumb" alt="captcha-thumb" />
            </div>
          </div>
          <div class="ft-captcha-slider-wrap">
            <div class="ft-captcha-slider-track">
              <div class="ft-captcha-slider-fill"></div>
              <div class="ft-captcha-slider-knob">↔</div>
            </div>
          </div>
          <div class="ft-captcha-tip">将滑块拖到合适位置以旋转图块</div>
        </div>
        <div class="ft-captcha-actions">
          <button type="button" class="ft-captcha-refresh">刷新</button>
          <button type="button" class="ft-captcha-cancel">取消</button>
          <button type="button" class="ft-captcha-confirm">确认验证</button>
        </div>
      </div>
    `;

    document.body.appendChild(overlay);

    const imageEl = overlay.querySelector('.ft-captcha-image');
    const thumbEl = overlay.querySelector('.ft-captcha-thumb');
    const thumbWrap = overlay.querySelector('.ft-captcha-thumb-wrap');
    const trackEl = overlay.querySelector('.ft-captcha-slider-track');
    const fillEl = overlay.querySelector('.ft-captcha-slider-fill');
    const knobEl = overlay.querySelector('.ft-captcha-slider-knob');
    const refreshBtn = overlay.querySelector('.ft-captcha-refresh');
    const cancelBtn = overlay.querySelector('.ft-captcha-cancel');
    const closeBtn = overlay.querySelector('.ft-captcha-close');
    const confirmBtn = overlay.querySelector('.ft-captcha-confirm');

    let dragMax = 1;
    let dragging = false;
    let startClientX = 0;
    let startLeft = 0;
    let sliderLeft = 0;
    let currentAngle = 0;
    let finished = false;

    function cleanup() {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      document.removeEventListener('touchmove', onTouchMove, { passive: false });
      document.removeEventListener('touchend', onUp);
      if (overlay.parentNode) {
        overlay.parentNode.removeChild(overlay);
      }
    }

    function setSliderByLeft(left) {
      sliderLeft = Math.max(0, Math.min(left, dragMax));
      const ratio = dragMax <= 0 ? 0 : sliderLeft / dragMax;
      currentAngle = Math.round(ratio * 360);
      knobEl.style.left = `${sliderLeft}px`;
      fillEl.style.width = `${sliderLeft + knobEl.offsetWidth / 2}px`;
      thumbWrap.style.transform = `translate(-50%, -50%) rotate(${currentAngle}deg)`;
    }

    function onMove(ev) {
      if (!dragging) {
        return;
      }
      const clientX = ev.clientX;
      setSliderByLeft(startLeft + (clientX - startClientX));
      ev.preventDefault();
    }

    function onTouchMove(ev) {
      if (!dragging) {
        return;
      }
      const t = ev.touches && ev.touches[0];
      if (!t) {
        return;
      }
      setSliderByLeft(startLeft + (t.clientX - startClientX));
      ev.preventDefault();
    }

    function onUp() {
      dragging = false;
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      document.removeEventListener('touchmove', onTouchMove, { passive: false });
      document.removeEventListener('touchend', onUp);
    }

    function bindDragStart(clientX) {
      dragging = true;
      startClientX = clientX;
      startLeft = sliderLeft;
      document.addEventListener('mousemove', onMove);
      document.addEventListener('mouseup', onUp);
      document.addEventListener('touchmove', onTouchMove, { passive: false });
      document.addEventListener('touchend', onUp);
    }

    function rejectAndClose(err) {
      if (finished) {
        return;
      }
      finished = true;
      cleanup();
      reject(err);
    }

    function resolveAndClose(result) {
      if (finished) {
        return;
      }
      finished = true;
      cleanup();
      resolve(result);
    }

    async function loadCaptcha(data) {
      captchaData = data;
      imageEl.src = normalizeImageSrc(data.imageBase64);
      thumbEl.src = normalizeImageSrc(data.thumbBase64);
      const size = Number(data.thumbSize || 84);
      thumbWrap.style.width = `${size}px`;
      thumbWrap.style.height = `${size}px`;
      await new Promise((r) => setTimeout(r, 0));
      dragMax = Math.max(trackEl.clientWidth - knobEl.clientWidth, 1);
      setSliderByLeft(0);
    }

    knobEl.addEventListener('mousedown', (ev) => {
      bindDragStart(ev.clientX);
      ev.preventDefault();
    });
    knobEl.addEventListener(
      'touchstart',
      (ev) => {
        const t = ev.touches && ev.touches[0];
        if (!t) {
          return;
        }
        bindDragStart(t.clientX);
        ev.preventDefault();
      },
      { passive: false }
    );

    cancelBtn.addEventListener('click', () => rejectAndClose(new Error('用户取消验证码')));
    closeBtn.addEventListener('click', () => rejectAndClose(new Error('用户取消验证码')));

    refreshBtn.addEventListener('click', async () => {
      try {
        const data = await getRotateCaptcha();
        await loadCaptcha(data);
      } catch (err) {
        rejectAndClose(err);
      }
    });

    confirmBtn.addEventListener('click', () => {
      resolveAndClose({
        captchaId: captchaData.id,
        captchaCode: String(currentAngle),
        captchaProtocol: 2
      });
    });

    overlay.addEventListener('click', (ev) => {
      if (ev.target === overlay) {
        rejectAndClose(new Error('用户取消验证码'));
      }
    });

    loadCaptcha(captchaData).catch((err) => rejectAndClose(err));
  });
}

async function postForm(path, formData, token) {
  const body = new URLSearchParams();
  Object.entries(formData).forEach(([k, v]) => {
    if (v !== undefined && v !== null) {
      body.append(k, String(v));
    }
  });

  const headers = {
    'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8'
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const payload = await requestJson(buildUrl(path), {
    method: 'POST',
    headers,
    body
  });
  return unwrapApi(payload);
}

async function postFormWithOptionalRotateCaptcha(path, formData, token) {
  // 1) rotate 关闭：优先走“图片验证码 protocol=3”（如果开关开启）；否则直接提交
  if (!getRotateCaptchaEnabled()) {
    if (!getImageCaptchaEnabled()) {
      return postForm(path, formData, token);
    }
    const captcha = await openImageCaptcha();
    return postForm(
      path,
      {
        ...formData,
        captchaId: captcha.captchaId,
        captchaCode: captcha.captchaCode,
        captchaProtocol: captcha.captchaProtocol
      },
      token
    );
  }

  // 2) rotate 开启：优先直提（提供与后端 loginCaptcha 开关联调的体验）
  try {
    return await postForm(path, formData, token);
  } catch (err) {
    // 3) 失败后再弹 rotate 兜底并重试一次
    const captcha = await openRotateCaptcha();
    return postForm(
      path,
      {
        ...formData,
        captchaId: captcha.captchaId,
        captchaCode: captcha.captchaCode,
        captchaProtocol: captcha.captchaProtocol
      },
      token
    );
  }
}

async function getCurrentUser(token) {
  const headers = {};
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const payload = await requestJson(buildUrl('/api/user/current'), {
    method: 'GET',
    headers
  });
  return unwrapApi(payload);
}

function renderJson(node, data) {
  node.textContent = JSON.stringify(data, null, 2);
}

function setMessage(node, text, ok) {
  node.textContent = text || '';
  node.classList.toggle('ok', Boolean(ok));
}

window.FrontTest = {
  getBackendBaseUrl,
  setBackendBaseUrl,
  normalizeBaseUrl,
  getRotateCaptchaEnabled,
  setRotateCaptchaEnabled,
  getImageCaptchaEnabled,
  setImageCaptchaEnabled,
  __debugGetStorage() {
    return {
      backendBaseUrlRaw: localStorage.getItem(STORAGE_BACKEND),
      backendBaseUrlFinal: getBackendBaseUrl(),
      token: localStorage.getItem(STORAGE_TOKEN),
      rotateCaptchaEnabledRaw: localStorage.getItem(STORAGE_ROTATE_CAPTCHA_ENABLED),
      imageCaptchaEnabledRaw: localStorage.getItem(STORAGE_IMAGE_CAPTCHA_ENABLED)
    };
  },
  saveToken,
  getToken,
  getRotateCaptcha,
  openRotateCaptcha,
  getImageCaptcha,
  openImageCaptcha,
  postForm,
  postFormWithOptionalRotateCaptcha,
  getCurrentUser,
  renderJson,
  setMessage
};
