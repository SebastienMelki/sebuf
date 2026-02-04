import React, { useState, useCallback } from "react";
import {
  ScrollView,
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Platform,
  SafeAreaView,
} from "react-native";
import { StatusBar } from "expo-status-bar";
import {
  TaskServiceClient,
  ValidationError,
  ApiError,
  type Task,
} from "./generated/proto/task_service_client";

// Platform-aware base URL
const BASE_URL = Platform.select({
  android: "http://10.0.2.2:3000",
  default: "http://localhost:3000",
});

// Shared API key for the demo
const API_KEY = "550e8400-e29b-41d4-a716-446655440000";

// Create client with service-level header
const client = new TaskServiceClient(BASE_URL, {
  apiKey: API_KEY,
});

// --- Collapsible Section Component ---

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(false);
  return (
    <View style={styles.section}>
      <TouchableOpacity
        style={styles.sectionHeader}
        onPress={() => setOpen(!open)}
      >
        <Text style={styles.sectionTitle}>
          {open ? "v" : ">"} {title}
        </Text>
      </TouchableOpacity>
      {open && <View style={styles.sectionBody}>{children}</View>}
    </View>
  );
}

// --- Result Display ---

function ResultBox({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.resultBox}>
      <Text style={styles.resultLabel}>{label}</Text>
      <Text style={styles.resultValue}>{value}</Text>
    </View>
  );
}

// --- Main App ---

export default function App() {
  const [results, setResults] = useState<Record<string, string>>({});

  const setResult = useCallback((key: string, value: string) => {
    setResults((prev) => ({ ...prev, [key]: value }));
  }, []);

  // --- CRUD Handlers ---

  const listTasks = async () => {
    try {
      const resp = await client.listTasks({
        status: "",
        limit: 0,
        offset: 0,
      });
      setResult(
        "list",
        `Found ${resp.total} tasks:\n${resp.tasks.map((t) => `  ${t.id}: ${t.title} [${t.status}]`).join("\n")}`
      );
    } catch (e: unknown) {
      setResult("list", `Error: ${e}`);
    }
  };

  const getTask = async () => {
    try {
      const task = await client.getTask({ id: "task-1" });
      setResult(
        "get",
        `${task.title}\n  Priority: ${task.priority}\n  Status: ${task.status}\n  Labels: ${task.labels.map((l) => l.name).join(", ")}\n  Metadata: ${JSON.stringify(task.metadata)}`
      );
    } catch (e: unknown) {
      setResult("get", `Error: ${e}`);
    }
  };

  const createTask = async () => {
    try {
      const task = await client.createTask(
        {
          title: "New task from React Native",
          description: "Created via the RN demo app",
          priority: "PRIORITY_MEDIUM",
          labels: [{ name: "mobile", color: "#06b6d4" }],
          metadata: { source: "rn-app" },
          dueDate: "2026-06-01",
        },
        { requestId: `req-${Date.now()}` }
      );
      setResult("create", `Created: ${task.id} - ${task.title}`);
    } catch (e: unknown) {
      setResult("create", `Error: ${e}`);
    }
  };

  const updateTask = async () => {
    try {
      const task = await client.updateTask({
        id: "task-3",
        title: "Design landing page (updated)",
        description: "Updated via RN app",
        priority: "PRIORITY_HIGH",
        status: "TASK_STATUS_IN_PROGRESS",
        labels: [
          { name: "frontend", color: "#10b981" },
          { name: "urgent", color: "#ef4444" },
        ],
        metadata: { updatedBy: "rn-app" },
      });
      setResult("update", `Updated: ${task.id} - ${task.title} [${task.status}]`);
    } catch (e: unknown) {
      setResult("update", `Error: ${e}`);
    }
  };

  const deleteTask = async () => {
    try {
      const resp = await client.deleteTask({ id: "task-4" });
      setResult("delete", `Deleted: success=${resp.success}`);
    } catch (e: unknown) {
      setResult("delete", `Error: ${e}`);
    }
  };

  // --- Query & Filter Handlers ---

  const filterByStatus = async () => {
    try {
      const resp = await client.listTasks({
        status: "todo",
        limit: 0,
        offset: 0,
      });
      setResult(
        "filter-status",
        `TODO tasks (${resp.total}):\n${resp.tasks.map((t) => `  ${t.id}: ${t.title}`).join("\n")}`
      );
    } catch (e: unknown) {
      setResult("filter-status", `Error: ${e}`);
    }
  };

  const paginate = async () => {
    try {
      const page1 = await client.listTasks({
        status: "",
        limit: 2,
        offset: 0,
      });
      const page2 = await client.listTasks({
        status: "",
        limit: 2,
        offset: 2,
      });
      setResult(
        "paginate",
        `Page 1 (${page1.tasks.length} of ${page1.total}):\n${page1.tasks.map((t) => `  ${t.id}: ${t.title}`).join("\n")}\n\nPage 2 (${page2.tasks.length} of ${page2.total}):\n${page2.tasks.map((t) => `  ${t.id}: ${t.title}`).join("\n")}`
      );
    } catch (e: unknown) {
      setResult("paginate", `Error: ${e}`);
    }
  };

  // --- Validation Handlers ---

  const triggerValidation = async () => {
    try {
      await client.createTask(
        {
          title: "", // min_len: 1 will fail
          description: "",
          priority: "PRIORITY_LOW",
          labels: [],
          metadata: {},
        },
        { requestId: `req-${Date.now()}` }
      );
      setResult("validation", "Unexpected success - should have failed");
    } catch (e: unknown) {
      if (e instanceof ValidationError) {
        setResult(
          "validation",
          `ValidationError caught!\nViolations:\n${e.violations.map((v) => `  ${v.field}: ${v.description}`).join("\n")}`
        );
      } else {
        setResult("validation", `Other error: ${e}`);
      }
    }
  };

  // --- Error Handling ---

  const trigger404 = async () => {
    try {
      await client.getTask({ id: "nonexistent-id" });
      setResult("error-404", "Unexpected success");
    } catch (e: unknown) {
      if (e instanceof ApiError) {
        let detail = `ApiError: status=${e.statusCode}\n  body: ${e.body}`;
        try {
          const parsed = JSON.parse(e.body);
          if (parsed.taskId) {
            detail += `\n\nParsed TaskNotFoundError:\n  taskId: ${parsed.taskId}\n  message: ${parsed.message}`;
          }
        } catch {
          // body wasn't JSON
        }
        setResult("error-404", detail);
      } else {
        setResult("error-404", `Other error: ${e}`);
      }
    }
  };

  // --- Advanced Features ---

  const testAbort = async () => {
    const controller = new AbortController();
    // Abort immediately
    controller.abort();
    try {
      await client.listTasks(
        { status: "", limit: 0, offset: 0 },
        { signal: controller.signal }
      );
      setResult("abort", "Unexpected success");
    } catch (e: unknown) {
      setResult("abort", `Aborted: ${(e as Error).name} - ${(e as Error).message}`);
    }
  };

  const testCustomFetch = async () => {
    const logs: string[] = [];
    const loggingClient = new TaskServiceClient(BASE_URL, {
      apiKey: API_KEY,
      fetch: async (input, init) => {
        const url = typeof input === "string" ? input : (input as Request).url;
        logs.push(`>> ${init?.method ?? "GET"} ${url}`);
        const start = Date.now();
        const resp = await fetch(input, init);
        logs.push(`<< ${resp.status} (${Date.now() - start}ms)`);
        return resp;
      },
    });
    try {
      await loggingClient.listTasks({ status: "", limit: 0, offset: 0 });
      setResult("custom-fetch", `Custom fetch with logging:\n${logs.join("\n")}`);
    } catch (e: unknown) {
      setResult("custom-fetch", `Error: ${e}\nLogs:\n${logs.join("\n")}`);
    }
  };

  const testUnwrap = async () => {
    try {
      const tasks: Task[] = await client.getTasksByLabel({ label: "backend" });
      setResult(
        "unwrap",
        `getTasksByLabel returns Task[] directly (unwrapped):\n  Type: Array (length=${tasks.length})\n${tasks.map((t) => `  ${t.id}: ${t.title}`).join("\n")}`
      );
    } catch (e: unknown) {
      setResult("unwrap", `Error: ${e}`);
    }
  };

  return (
    <SafeAreaView style={styles.container}>
      <StatusBar style="light" />
      <ScrollView style={styles.scroll} contentContainerStyle={styles.content}>
        <Text style={styles.heading}>sebuf RN Client Demo</Text>
        <Text style={styles.subheading}>
          TaskService API - all protoc-gen-ts-client features
        </Text>

        {/* CRUD */}
        <Section title="CRUD Operations">
          <TouchableOpacity style={styles.button} onPress={listTasks}>
            <Text style={styles.buttonText}>List All Tasks</Text>
          </TouchableOpacity>
          {results.list && <ResultBox label="ListTasks" value={results.list} />}

          <TouchableOpacity style={styles.button} onPress={getTask}>
            <Text style={styles.buttonText}>Get Task (task-1)</Text>
          </TouchableOpacity>
          {results.get && <ResultBox label="GetTask" value={results.get} />}

          <TouchableOpacity style={styles.button} onPress={createTask}>
            <Text style={styles.buttonText}>Create Task</Text>
          </TouchableOpacity>
          {results.create && (
            <ResultBox label="CreateTask" value={results.create} />
          )}

          <TouchableOpacity style={styles.button} onPress={updateTask}>
            <Text style={styles.buttonText}>Update Task (task-3)</Text>
          </TouchableOpacity>
          {results.update && (
            <ResultBox label="UpdateTask" value={results.update} />
          )}

          <TouchableOpacity style={styles.button} onPress={deleteTask}>
            <Text style={styles.buttonText}>Delete Task (task-4)</Text>
          </TouchableOpacity>
          {results.delete && (
            <ResultBox label="DeleteTask" value={results.delete} />
          )}
        </Section>

        {/* Query & Filters */}
        <Section title="Query & Filters">
          <TouchableOpacity style={styles.button} onPress={filterByStatus}>
            <Text style={styles.buttonText}>Filter by Status (todo)</Text>
          </TouchableOpacity>
          {results["filter-status"] && (
            <ResultBox label="Filter" value={results["filter-status"]} />
          )}

          <TouchableOpacity style={styles.button} onPress={paginate}>
            <Text style={styles.buttonText}>Paginate (limit=2)</Text>
          </TouchableOpacity>
          {results.paginate && (
            <ResultBox label="Paginate" value={results.paginate} />
          )}
        </Section>

        {/* Headers */}
        <Section title="Headers">
          <Text style={styles.infoText}>
            Service header X-API-Key is set at client init:{"\n"}
            {API_KEY.substring(0, 8)}...
          </Text>
          <Text style={styles.infoText}>
            Method header X-Request-ID is passed per-call on CreateTask.
          </Text>
          <TouchableOpacity style={styles.button} onPress={createTask}>
            <Text style={styles.buttonText}>
              Create Task (with X-Request-ID)
            </Text>
          </TouchableOpacity>
          {results.create && (
            <ResultBox label="CreateTask" value={results.create} />
          )}
        </Section>

        {/* Validation */}
        <Section title="Validation Errors">
          <TouchableOpacity style={styles.button} onPress={triggerValidation}>
            <Text style={styles.buttonText}>
              Create with Empty Title (triggers validation)
            </Text>
          </TouchableOpacity>
          {results.validation && (
            <ResultBox label="Validation" value={results.validation} />
          )}
        </Section>

        {/* Error Handling */}
        <Section title="Error Handling">
          <TouchableOpacity style={styles.button} onPress={trigger404}>
            <Text style={styles.buttonText}>
              Get Nonexistent Task (triggers 404)
            </Text>
          </TouchableOpacity>
          {results["error-404"] && (
            <ResultBox label="404 Error" value={results["error-404"]} />
          )}
        </Section>

        {/* Advanced */}
        <Section title="Advanced Features">
          <TouchableOpacity style={styles.button} onPress={testAbort}>
            <Text style={styles.buttonText}>AbortController Cancellation</Text>
          </TouchableOpacity>
          {results.abort && (
            <ResultBox label="Abort" value={results.abort} />
          )}

          <TouchableOpacity style={styles.button} onPress={testCustomFetch}>
            <Text style={styles.buttonText}>Custom Fetch with Logging</Text>
          </TouchableOpacity>
          {results["custom-fetch"] && (
            <ResultBox label="Custom Fetch" value={results["custom-fetch"]} />
          )}

          <TouchableOpacity style={styles.button} onPress={testUnwrap}>
            <Text style={styles.buttonText}>
              Unwrap Response (getTasksByLabel)
            </Text>
          </TouchableOpacity>
          {results.unwrap && (
            <ResultBox label="Unwrap" value={results.unwrap} />
          )}
        </Section>

        <View style={styles.footer} />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#0f172a",
  },
  scroll: {
    flex: 1,
  },
  content: {
    padding: 16,
  },
  heading: {
    fontSize: 24,
    fontWeight: "bold",
    color: "#f1f5f9",
    marginTop: 8,
    marginBottom: 4,
  },
  subheading: {
    fontSize: 14,
    color: "#94a3b8",
    marginBottom: 20,
  },
  section: {
    marginBottom: 12,
    borderRadius: 8,
    backgroundColor: "#1e293b",
    overflow: "hidden",
  },
  sectionHeader: {
    padding: 14,
  },
  sectionTitle: {
    fontSize: 16,
    fontWeight: "600",
    color: "#e2e8f0",
  },
  sectionBody: {
    padding: 12,
    paddingTop: 0,
  },
  button: {
    backgroundColor: "#3b82f6",
    paddingVertical: 10,
    paddingHorizontal: 14,
    borderRadius: 6,
    marginBottom: 8,
  },
  buttonText: {
    color: "#ffffff",
    fontSize: 14,
    fontWeight: "500",
  },
  infoText: {
    color: "#94a3b8",
    fontSize: 13,
    marginBottom: 8,
    lineHeight: 18,
  },
  resultBox: {
    backgroundColor: "#0f172a",
    borderRadius: 6,
    padding: 10,
    marginBottom: 8,
  },
  resultLabel: {
    fontSize: 12,
    fontWeight: "600",
    color: "#60a5fa",
    marginBottom: 4,
  },
  resultValue: {
    fontSize: 12,
    color: "#cbd5e1",
    fontFamily: Platform.select({ ios: "Menlo", default: "monospace" }),
    lineHeight: 18,
  },
  footer: {
    height: 40,
  },
});
