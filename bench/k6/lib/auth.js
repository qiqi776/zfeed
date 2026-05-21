import { check2xx, jsonField, postJSON } from "./http.js";
import { benchConfig } from "../config.js";

function uniqueMobile(index) {
  const seed = (Date.now() + index).toString().slice(-10).padStart(10, "0");
  return `+861${seed}`;
}

export function registerAndLogin(baseURL, template, index) {
  const mobile = __ENV.BENCH_STATIC_USERS === "1" ? template.mobile : uniqueMobile(index);
  const password = template.password || benchConfig.defaultPassword;
  const nickname = `${template.nickname || "bench_user"}_${Date.now()}_${index}`;

  const registerRes = postJSON(
    baseURL,
    "/v1/users",
    {
      mobile,
      password,
      nickname,
      avatar: template.avatar || "https://example.com/bench/avatar.png",
      bio: template.bio || "bench generated user",
      gender: Number(template.gender || 1),
      email: `bench_${Date.now()}_${index}@example.com`,
      birthday: Number(template.birthday || 946684800),
    },
    "",
    { name: "user_register", module: "user", kind: "write", auth: "none" },
  );
  checkNonFatalRegister(registerRes);

  const loginRes = postJSON(
    baseURL,
    "/v1/login",
    { mobile, password },
    "",
    { name: "user_login", module: "user", kind: "write", auth: "none" },
  );
  check2xx(loginRes, "user_login");

  return {
    mobile,
    password,
    token: String(jsonField(loginRes, "token", jsonField(registerRes, "token", ""))),
    userId: Number(jsonField(loginRes, "user_id", jsonField(registerRes, "user_id", 0))),
  };
}

function checkNonFatalRegister(res) {
  if (res.status === 409) {
    return;
  }
  check2xx(res, "user_register");
}

