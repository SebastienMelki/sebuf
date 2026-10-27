// Hand-written. Pins, through the TypeScript compiler, what the generated
// wire_presence.ts lets a caller write: the generated client sends
// JSON.stringify(request), so these types are the wire.
import type { FileKind, PresenceProbe } from "./wire_presence.js";

// (sebuf.http.enum_value): the wire strings are the type.
export const jpeg: FileKind = "JPEG";
export const png: FileKind = "PNG";
// A value without enum_value keeps its proto name.
export const unspecified: FileKind = "FILE_KIND_UNSPECIFIED";
// @ts-expect-error the proto name of an annotated value is not on the wire
export const protoName: FileKind = "FILE_KIND_JPEG";

// An unset optional scalar or message field is an absent key; an unset
// repeated field and a non-optional scalar are always written; a nullable
// field is written, as its value or as null.
export const minimal: PresenceProbe = {
  name: "",
  tags: [],
  places: [],
  kind: "JPEG",
  kinds: [],
  nullable_note: null,
  nullable_quality: null,
};

export const full: PresenceProbe = {
  ...minimal,
  note: "n",
  quality: 0,
  location: { city: "Paris" },
  extra: { city: "", state: "" },
  optional_kind: "MP4",
};

// @ts-expect-error a plain optional scalar cannot be null
export const nullNote: PresenceProbe = { ...minimal, note: null };

// @ts-expect-error a plain optional message field cannot be null
export const nullLocation: PresenceProbe = { ...minimal, location: null };

// @ts-expect-error a repeated field cannot be left out
export const noTags: PresenceProbe = { name: "", places: [], kind: "JPEG", kinds: [], nullable_note: null, nullable_quality: null };

// @ts-expect-error a nullable field must be given, as a value or null
export const noNullable: PresenceProbe = { name: "", tags: [], places: [], kind: "JPEG", kinds: [] };
