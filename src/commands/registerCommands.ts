import * as vscode from "vscode";
import { ConfigService } from "../config/configService";
import { DeinsCompleteLifecycle } from "../core/lifecycle";
import { Logger } from "../logging/logger";

export function registerCommands(
  config: ConfigService,
  lifecycle: DeinsCompleteLifecycle,
  logger: Logger,
): vscode.Disposable[] {
  return [
    vscode.commands.registerCommand("deinscomplete.enable", async () => {
      await config.setEnabled(true);
      lifecycle.refresh();
      logger.info("DeinsComplete enabled.");
    }),
    vscode.commands.registerCommand("deinscomplete.disable", async () => {
      await config.setEnabled(false);
      lifecycle.refresh();
      logger.info("DeinsComplete disabled.");
    }),
    vscode.commands.registerCommand("deinscomplete.showLogs", () => logger.show()),
  ];
}
