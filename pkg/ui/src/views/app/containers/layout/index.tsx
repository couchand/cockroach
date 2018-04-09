import React from "react";
import _ from "lodash";
import DocumentTitle from "react-document-title";
import { RouterState } from "react-router";
import { StickyContainer } from "react-sticky";

import NavigationBar from "src/views/app/components/layoutSidebar";
import TimeWindowManager from "src/views/app/containers/timewindow";
import AlertBanner from "src/views/app/containers/alertBanner";
import { TitledComponent } from "src/views/shared/components/titledComponent";

function isTitledComponent<P, S>(obj: Object | TitledComponent<P, S>): obj is TitledComponent<P, S> {
  return obj && _.isFunction((obj as TitledComponent<P, S>).title);
}

/**
 * Defines the main layout of all admin ui pages. This includes static
 * navigation bars and footers which should be present on every page.
 *
 * Individual pages provide their content via react-router.
 */
export default class extends React.Component<RouterState, {}> {
  render() {
    const child = React.Children.only(this.props.children);

    let title: string;
    let header: React.ReactElement<any>;

    if (isTitledComponent(child)) {
      title = child.title(this.props);
      header = child.header(this.props);
    }

    const pageTitle = title ? title + " | Cockroach Console" : "Cockroach Console";

    return (
      <DocumentTitle title={pageTitle}>
        <div>
          <TimeWindowManager/>
          <AlertBanner/>
          <NavigationBar/>
          <StickyContainer className="page">
            {
              // TODO(mrtracy): The title can be moved down to individual pages,
              // it is not always the top element on the page (for example, on
              // pages with a back button).
              !!header
                ? <section className="section">{ header }</section>
                : null
            }
            { child }
          </StickyContainer>
        </div>
      </DocumentTitle>
    );
  }
}
