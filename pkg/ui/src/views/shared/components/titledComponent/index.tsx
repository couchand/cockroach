import React from "react";
import { RouterState } from "react-router";

export abstract class TitledComponent<Props, State = {}> extends React.Component<Props, State> {
  abstract title(routeProps: RouterState): string;

  header(routeProps: RouterState): React.ReactElement<any> {
    return <h1>{ this.title(routeProps) }</h1>;
  }
}
