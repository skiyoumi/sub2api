// Only selection IDs are stored in the tab session. Prompts and images are
// always read from authenticated, expiring server records.
interface StudioSelection { model: string; taskID: string }
interface StudioSession { keyID: string; selections: Record<string, StudioSelection> }
const prefix = 'image-studio-selection:'

export function readStudioSession(userID: number): StudioSession {
  try {
    const raw = JSON.parse(sessionStorage.getItem(`${prefix}${userID}`) || 'null')
    if (typeof raw?.keyID === 'string' && raw.selections && typeof raw.selections === 'object') return raw
  } catch { /* Storage may be unavailable. The server remains the source of truth. */ }
  return { keyID: '', selections: {} }
}

export function saveStudioSelection(userID: number, keyID: string, selection?: StudioSelection) {
  if (!userID) return
  const session = readStudioSession(userID)
  session.keyID = keyID
  if (selection && keyID) session.selections[keyID] = selection
  try { sessionStorage.setItem(`${prefix}${userID}`, JSON.stringify(session)) } catch { /* Best effort. */ }
}
