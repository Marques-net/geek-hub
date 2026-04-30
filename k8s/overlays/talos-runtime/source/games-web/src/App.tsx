import { ChessGame } from "./games/chess/ChessGame";
import { TicTacToeGame } from "./games/tictactoe/TicTacToeGame";

type GameModule = {
  slug: string;
  title: string;
  render: () => JSX.Element;
};

const gameModules: GameModule[] = [
  {
    slug: "chess",
    title: "Xadrez Online",
    render: () => <ChessGame />
  },
  {
    slug: "tictactoe",
    title: "Jogo da Velha",
    render: () => <TicTacToeGame />
  }
];

const resolveGameSlug = (): string => {
  const [, first, second] = window.location.pathname.split("/");

  if (first === "games" && second) {
    return second.toLowerCase();
  }

  if (first === "chess") {
    return "chess";
  }
  if (first === "tictactoe" || first === "jogo-da-velha") {
    return "tictactoe";
  }
  return "chess";
};

export default function App() {
  const gameSlug = resolveGameSlug();
  const gameModule = gameModules.find((candidate) => candidate.slug === gameSlug);

  if (!gameModule) {
    return (
      <main className="app-shell">
        <section className="panel lobby-panel status-panel">
          <div className="eyebrow">Geek Hub</div>
          <h1>Jogo não encontrado</h1>
          <p className="lead">
            O módulo solicitado ainda não foi registrado no host de jogos.
          </p>
          <button className="button" onClick={() => window.location.assign("/")}>
            Voltar ao portal
          </button>
        </section>
      </main>
    );
  }

  document.title = `${gameModule.title} | Geek Hub`;
  return gameModule.render();
}
