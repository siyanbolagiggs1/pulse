"use client";
import * as React from "react";
import type { ToastProps, ToastActionElement } from "@/components/ui/toast";

const TOAST_LIMIT = 3;
const TOAST_REMOVE_DELAY = 4000;

type ToasterToast = ToastProps & {
  id: string;
  title?: React.ReactNode;
  description?: React.ReactNode;
  action?: ToastActionElement;
};

let count = 0;
function genId() { return `toast-${++count}`; }

type State = { toasts: ToasterToast[] };

const listeners: Array<(s: State) => void> = [];
let memState: State = { toasts: [] };

function dispatch(update: (s: State) => State) {
  memState = update(memState);
  listeners.forEach((l) => l(memState));
}

function toast({ title, description, variant, ...props }: Omit<ToasterToast, "id">) {
  const id = genId();
  dispatch((s) => ({
    toasts: [{ id, title, description, variant, open: true, ...props }, ...s.toasts].slice(0, TOAST_LIMIT),
  }));
  setTimeout(() => {
    dispatch((s) => ({
      toasts: s.toasts.map((t) => (t.id === id ? { ...t, open: false } : t)),
    }));
  }, TOAST_REMOVE_DELAY);
  return id;
}

function useToast() {
  const [state, setState] = React.useState<State>(memState);
  React.useEffect(() => {
    listeners.push(setState);
    return () => { listeners.splice(listeners.indexOf(setState), 1); };
  }, []);
  return {
    toasts: state.toasts,
    toast,
    dismiss: (id: string) =>
      dispatch((s) => ({ toasts: s.toasts.map((t) => (t.id === id ? { ...t, open: false } : t)) })),
  };
}

export { useToast, toast };
