import { useEffect, useRef, useState } from "react";

import { getGoogleClientId } from "../lib/config";
import { AuthSession } from "../types";

interface GoogleSignInButtonProps {
  busy: boolean;
  onSuccess: (session: AuthSession) => void;
  onError: (message: string) => void;
}

interface GoogleJwtPayload {
  sub?: string;
  name?: string;
  given_name?: string;
  email?: string;
  picture?: string;
}

const GIS_SCRIPT_ID = "google-identity-services";

const decodeBase64Url = (value: string): string => {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  const binary = window.atob(padded);
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
  return new TextDecoder().decode(bytes);
};

const toAuthSession = (credential: string): AuthSession => {
  const parts = credential.split(".");
  if (parts.length < 2) {
    throw new Error("Resposta inválida do Google Identity.");
  }

  const payload = JSON.parse(decodeBase64Url(parts[1])) as GoogleJwtPayload;
  if (!payload.sub || !payload.name) {
    throw new Error("Perfil do Google incompleto.");
  }

  return {
    provider: "google",
    sub: payload.sub,
    name: payload.name,
    givenName: payload.given_name ?? payload.name,
    email: payload.email ?? null,
    picture: payload.picture ?? null
  };
};

const loadGoogleIdentityScript = async (): Promise<void> => {
  if (window.google?.accounts.id) {
    return;
  }

  await new Promise<void>((resolve, reject) => {
    const existing = document.getElementById(GIS_SCRIPT_ID) as HTMLScriptElement | null;
    if (existing) {
      if (existing.dataset.loaded === "true") {
        resolve();
        return;
      }

      existing.addEventListener("load", () => resolve(), { once: true });
      existing.addEventListener("error", () => reject(new Error("Falha ao carregar Google Identity.")), {
        once: true
      });
      return;
    }

    const script = document.createElement("script");
    script.id = GIS_SCRIPT_ID;
    script.src = "https://accounts.google.com/gsi/client";
    script.async = true;
    script.defer = true;
    script.onload = () => {
      script.dataset.loaded = "true";
      resolve();
    };
    script.onerror = () => reject(new Error("Falha ao carregar Google Identity."));
    document.head.appendChild(script);
  });
};

export const GoogleSignInButton = ({ busy, onSuccess, onError }: GoogleSignInButtonProps) => {
  const buttonRef = useRef<HTMLDivElement | null>(null);
  const [status, setStatus] = useState<"idle" | "loading" | "ready" | "missing">("idle");
  const clientId = getGoogleClientId();

  useEffect(() => {
    if (!clientId) {
      setStatus("missing");
      return;
    }

    let cancelled = false;
    setStatus("loading");

    void loadGoogleIdentityScript()
      .then(() => {
        if (cancelled || !buttonRef.current || !window.google?.accounts.id) {
          return;
        }

        window.google.accounts.id.initialize({
          client_id: clientId,
          ux_mode: "popup",
          callback: (response) => {
            try {
              onSuccess(toAuthSession(response.credential));
            } catch (error) {
              onError(error instanceof Error ? error.message : "Falha ao validar o login Google.");
            }
          }
        });

        buttonRef.current.innerHTML = "";
        window.google.accounts.id.renderButton(buttonRef.current, {
          theme: "outline",
          size: "large",
          type: "standard",
          shape: "pill",
          text: "signin_with",
          logo_alignment: "left",
          width: 320,
          locale: navigator.language || "pt-BR"
        });
        setStatus("ready");
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }

        setStatus("missing");
        onError(error instanceof Error ? error.message : "Falha ao carregar o Google Identity.");
      });

    return () => {
      cancelled = true;
    };
  }, [clientId, onError, onSuccess]);

  if (!clientId) {
    return (
      <div className="google-button-shell google-button-missing">
        <button className="button button-secondary" disabled>
          Entrar com Google
        </button>
        <small>
          Defina <code>VITE_GOOGLE_CLIENT_ID</code> para habilitar o login Google neste
          ambiente.
        </small>
      </div>
    );
  }

  return (
    <div className={`google-button-shell${busy ? " google-button-shell-busy" : ""}`}>
      {status === "loading" ? <small>Carregando Google Identity...</small> : null}
      <div ref={buttonRef} className="google-button-target" />
    </div>
  );
};
