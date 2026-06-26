export function isValidMobile(value: string) {
    const compact = value.trim().replace(/[ \-()]/g, "");
    return isCNRaw(compact) || isCNWith86(compact) || isCNWithPlus86(compact) || isE164(compact);
}

function isCNRaw(value: string) {
    return value.length === 11 && value[0] === "1" && digitsOnly(value);
}

function isCNWith86(value: string) {
    return value.length === 13 && value.startsWith("86") && isCNRaw(value.slice(2));
}

function isCNWithPlus86(value: string) {
    return value.length === 14 && value.startsWith("+86") && isCNRaw(value.slice(3));
}

function isE164(value: string) {
    if (value.length < 8 || value.length > 16 || value[0] !== "+") {
        return false;
    }

    const firstDigit = value[1];
    return firstDigit >= "1" && firstDigit <= "9" && digitsOnly(value.slice(1));
}

function digitsOnly(value: string) {
    return /^\d+$/.test(value);
}
