/** Build the native client link used by QQ's official group sharing page. */
export function buildQqGroupJoinUrl(groupNumber: string, authKey: string, mobile: boolean): string {
  if (mobile) {
    const params = new URLSearchParams({
      src_type: 'app',
      version: '1',
      uin: groupNumber,
      card_type: 'group',
      wSourceSubID: '1027',
      authSig: authKey,
    })
    return `mqqapi://card/show_pslcard?${params}`
  }

  // Desktop QQ expects a hex-encoded JSON payload with a fresh millisecond timestamp.
  const payload = JSON.stringify({ groupUin: groupNumber, timeStamp: Date.now(), authKey, auth: '' })
  const param = Array.from(new TextEncoder().encode(payload), byte => byte.toString(16).padStart(2, '0'))
    .join('')
    .toUpperCase()
  return `tencent://groupwpa/?subcmd=all&param=${param}&jump_from=`
}
