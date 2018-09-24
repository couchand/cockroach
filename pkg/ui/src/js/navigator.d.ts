import { Middleware } from "redux";
import { PayloadAction } from "src/interfaces/action";

declare function setRouteParam(param: string, value: string): PayloadAction<{ param: string, value: string}>;
declare const navigatorMiddleware: Middleware;
