import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import {
  createNoteServiceRoutes,
  type NoteServiceHandler,
  type ServerContext,
  type Note,
  type RouteDescriptor,
  type Priority,
  type Status,
} from "./generated/proto/note_service_server.ts";

// ==========================================================================
// SSE log streaming
// ==========================================================================

const sseClients = new Set<ServerResponse>();

function broadcast(event: string, data: unknown) {
  const msg = `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
  for (const client of sseClients) {
    client.write(msg);
  }
}

function serverLog(message: string) {
  console.log(message);
  broadcast("log", { time: Date.now(), message });
}

function requestLog(method: string, path: string, status: number, duration: number) {
  broadcast("request", { time: Date.now(), method, path, status, duration });
}

// ==========================================================================
// Static file serving
// ==========================================================================

const __dirname = dirname(fileURLToPath(import.meta.url));
const indexHtml = readFileSync(join(__dirname, "public", "index.html"), "utf-8");

// ==========================================================================
// In-memory store
// ==========================================================================

const notes = new Map<string, Note>();
let nextId = 5;

function seedData() {
  const now = Date.now();
  notes.set("note-1", {
    id: "note-1",
    title: "Design API schema",
    content: "Define proto messages and service RPCs",
    priority: "PRIORITY_HIGH",
    status: "STATUS_DONE",
    tags: [
      { name: "backend", color: "#3b82f6" },
      { name: "design", color: "#8b5cf6" },
    ],
    metadata: { sprint: "12", team: "platform" },
    createdAt: new Date(now - 72 * 3600_000).toISOString(),
  });
  notes.set("note-2", {
    id: "note-2",
    title: "Write unit tests",
    content: "Cover edge cases for validation and error handling",
    priority: "PRIORITY_MEDIUM",
    status: "STATUS_IN_PROGRESS",
    tags: [
      { name: "backend", color: "#3b82f6" },
      { name: "testing", color: "#10b981" },
    ],
    metadata: { sprint: "12" },
    dueDate: "2025-12-31",
    createdAt: new Date(now - 48 * 3600_000).toISOString(),
  });
  notes.set("note-3", {
    id: "note-3",
    title: "Update documentation",
    content: "Add examples for all new features",
    priority: "PRIORITY_LOW",
    status: "STATUS_PENDING",
    tags: [{ name: "docs", color: "#f59e0b" }],
    metadata: {},
    createdAt: new Date(now - 24 * 3600_000).toISOString(),
  });
  notes.set("note-4", {
    id: "note-4",
    title: "Fix login bug",
    content: "Session expires too early on mobile",
    priority: "PRIORITY_URGENT",
    status: "STATUS_PENDING",
    tags: [
      { name: "backend", color: "#3b82f6" },
      { name: "bug", color: "#ef4444" },
    ],
    metadata: { reporter: "alice", severity: "critical" },
    dueDate: "2025-06-15",
    createdAt: new Date(now - 12 * 3600_000).toISOString(),
  });
}

// ==========================================================================
// Custom error for not-found
// ==========================================================================

class NotFoundError extends Error {
  resourceType: string;
  resourceId: string;

  constructor(resourceType: string, resourceId: string) {
    super(`${resourceType} '${resourceId}' not found`);
    this.name = "NotFoundError";
    this.resourceType = resourceType;
    this.resourceId = resourceId;
  }
}

// ==========================================================================
// Handler implementation
// ==========================================================================

const priorityOrder: Record<Priority, number> = {
  PRIORITY_UNSPECIFIED: 0,
  PRIORITY_LOW: 1,
  PRIORITY_MEDIUM: 2,
  PRIORITY_HIGH: 3,
  PRIORITY_URGENT: 4,
};

function matchStatus(noteStatus: Status, filter: string): boolean {
  const map: Record<string, Status> = {
    pending: "STATUS_PENDING",
    in_progress: "STATUS_IN_PROGRESS",
    done: "STATUS_DONE",
    archived: "STATUS_ARCHIVED",
  };
  return map[filter.toLowerCase()] === noteStatus;
}

function matchPriority(notePriority: Priority, filter: string): boolean {
  const map: Record<string, Priority> = {
    low: "PRIORITY_LOW",
    medium: "PRIORITY_MEDIUM",
    high: "PRIORITY_HIGH",
    urgent: "PRIORITY_URGENT",
  };
  return map[filter.toLowerCase()] === notePriority;
}

const handler: NoteServiceHandler = {
  async listNotes(_ctx: ServerContext, req) {
    let result = [...notes.values()];

    if (req.status) {
      result = result.filter((n) => matchStatus(n.status, req.status));
    }
    if (req.priority) {
      result = result.filter((n) => matchPriority(n.priority, req.priority));
    }

    // Sort
    switch (req.sort?.toLowerCase()) {
      case "title":
        result.sort((a, b) => a.title.localeCompare(b.title));
        break;
      case "priority":
        result.sort((a, b) => priorityOrder[b.priority] - priorityOrder[a.priority]);
        break;
      case "created_at":
        result.sort((a, b) => a.createdAt.localeCompare(b.createdAt));
        break;
      default:
        result.sort((a, b) => b.createdAt.localeCompare(a.createdAt));
    }

    const total = result.length;

    if (req.offset > 0 && req.offset < result.length) {
      result = result.slice(req.offset);
    } else if (req.offset > 0) {
      result = [];
    }
    if (req.limit > 0 && req.limit < result.length) {
      result = result.slice(0, req.limit);
    }

    return { notes: result, total };
  },

  async getNote(_ctx, req) {
    const note = notes.get(req.id);
    if (!note) throw new NotFoundError("note", req.id);
    return note;
  },

  async createNote(_ctx, req) {
    const id = `note-${nextId++}`;
    const note: Note = {
      id,
      title: req.title,
      content: req.content,
      priority: req.priority || "PRIORITY_UNSPECIFIED",
      status: "STATUS_PENDING",
      tags: req.tags ?? [],
      metadata: req.metadata ?? {},
      dueDate: req.dueDate,
      createdAt: new Date().toISOString(),
    };
    notes.set(id, note);
    serverLog(`Created note: ${id} - "${note.title}"`);
    return note;
  },

  async updateNote(_ctx, req) {
    const note = notes.get(req.id);
    if (!note) throw new NotFoundError("note", req.id);
    note.title = req.title;
    note.content = req.content;
    note.priority = req.priority;
    note.status = req.status;
    note.tags = req.tags ?? [];
    note.metadata = req.metadata ?? {};
    note.dueDate = req.dueDate;
    serverLog(`Updated note: ${note.id} - status=${note.status}`);
    return note;
  },

  async archiveNote(_ctx, req) {
    const note = notes.get(req.id);
    if (!note) throw new NotFoundError("note", req.id);
    note.status = "STATUS_ARCHIVED";
    serverLog(`Archived note: ${note.id}`);
    return note;
  },

  async deleteNote(_ctx, req) {
    if (!notes.has(req.id)) throw new NotFoundError("note", req.id);
    notes.delete(req.id);
    serverLog(`Deleted note: ${req.id}`);
    return { success: true };
  },

  async getNotesByTag(_ctx, req) {
    const result = [...notes.values()].filter((n) =>
      n.tags.some((t) => t.name.toLowerCase() === req.tag.toLowerCase()),
    );
    result.sort((a, b) => a.createdAt.localeCompare(b.createdAt));
    return result;
  },
};

// ==========================================================================
// Node HTTP adapter — bridges Node's http module to the Web Fetch API
// ==========================================================================

const routes = createNoteServiceRoutes(handler, {
  onError: (err, _req) => {
    if (err instanceof NotFoundError) {
      return new Response(
        JSON.stringify({ resourceType: err.resourceType, resourceId: err.resourceId }),
        { status: 404, headers: { "Content-Type": "application/json" } },
      );
    }
    const message = err instanceof Error ? err.message : String(err);
    return new Response(JSON.stringify({ message }), {
      status: 500,
      headers: { "Content-Type": "application/json" },
    });
  },
});

// Sort routes: static segments before parameterized ones (e.g., /notes/by-tag before /notes/{id})
routes.sort((a, b) => {
  const aParts = a.path.split("/");
  const bParts = b.path.split("/");
  const len = Math.max(aParts.length, bParts.length);
  for (let i = 0; i < len; i++) {
    const aParam = aParts[i]?.startsWith("{") ? 1 : 0;
    const bParam = bParts[i]?.startsWith("{") ? 1 : 0;
    if (aParam !== bParam) return aParam - bParam;
  }
  return 0;
});

function matchPath(pathname: string, pattern: string): boolean {
  const patternParts = pattern.split("/");
  const pathParts = pathname.split("/");
  if (patternParts.length !== pathParts.length) return false;
  return patternParts.every(
    (part, i) => (part.startsWith("{") && part.endsWith("}")) || part === pathParts[i],
  );
}

async function readBody(req: IncomingMessage): Promise<string> {
  return new Promise((resolve) => {
    let data = "";
    req.on("data", (chunk: Buffer) => (data += chunk.toString()));
    req.on("end", () => resolve(data));
  });
}

seedData();

const server = createServer(async (nodeReq, nodeRes) => {
  const url = new URL(nodeReq.url!, `http://localhost:3000`);

  // Serve browser UI
  if (url.pathname === "/" && nodeReq.method === "GET") {
    nodeRes.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
    nodeRes.end(indexHtml);
    return;
  }

  // SSE endpoint for live log streaming
  if (url.pathname === "/events" && nodeReq.method === "GET") {
    nodeRes.writeHead(200, {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
    });
    nodeRes.write("event: log\ndata: " + JSON.stringify({ time: Date.now(), message: "Connected to server log stream" }) + "\n\n");
    sseClients.add(nodeRes);
    nodeReq.on("close", () => sseClients.delete(nodeRes));
    return;
  }

  // API routes
  const t0 = Date.now();
  for (const route of routes) {
    if (nodeReq.method === route.method && matchPath(url.pathname, route.path)) {
      const headers = new Headers();
      for (const [key, value] of Object.entries(nodeReq.headers)) {
        if (value) headers.set(key, Array.isArray(value) ? value.join(", ") : value);
      }

      const hasBody = ["POST", "PUT", "PATCH"].includes(nodeReq.method!);
      const body = hasBody ? await readBody(nodeReq) : undefined;

      const request = new Request(url.toString(), {
        method: nodeReq.method,
        headers,
        body: body || undefined,
      });

      const response = await route.handler(request);
      const respHeaders: Record<string, string> = {};
      response.headers.forEach((v, k) => (respHeaders[k] = v));
      nodeRes.writeHead(response.status, respHeaders);
      nodeRes.end(await response.text());
      requestLog(nodeReq.method!, url.pathname, response.status, Date.now() - t0);
      return;
    }
  }

  nodeRes.writeHead(404, { "Content-Type": "application/json" });
  nodeRes.end(JSON.stringify({ message: "Not Found" }));
  requestLog(nodeReq.method!, url.pathname, 404, Date.now() - t0);
});

server.listen(3000, () => {
  console.log("TypeScript NoteService server running on http://localhost:3000");
  console.log("");
  console.log("  Browser UI:  http://localhost:3000");
  console.log("");
  console.log("  API Endpoints:");
  console.log("    GET    /api/v1/notes              - List notes");
  console.log("    GET    /api/v1/notes/{id}          - Get note");
  console.log("    POST   /api/v1/notes               - Create note");
  console.log("    PUT    /api/v1/notes/{id}           - Update note");
  console.log("    PATCH  /api/v1/notes/{id}/archive   - Archive note");
  console.log("    DELETE /api/v1/notes/{id}           - Delete note");
  console.log("    GET    /api/v1/notes/by-tag         - Get notes by tag");
  console.log("");
  console.log("  Headers: X-API-Key (uuid) + X-Tenant-ID (integer)");
  console.log("  4 seed notes pre-loaded");
});
