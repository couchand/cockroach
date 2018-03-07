import React from "react";

export default class NeedEnterpriseLicense extends React.Component {
  render() {
    return (
      <section className="section">
        <h1>Introducing the Node Map!</h1>
        <p>
          We've made the somewhat dubious decision to gate this feature behind
          a valid enterprise license, and you don't have one.  Sorry.
        </p>
      </section>
    );
  }
}
