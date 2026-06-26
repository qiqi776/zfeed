import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PageState } from "./PageState";

describe("PageState", () => {
    it("keeps long titles and descriptions from overflowing", () => {
        const longTitle = "很长的状态标题".repeat(12);
        const longDescription = "很长的状态描述".repeat(12);

        render(
            <PageState
                state="error"
                title={longTitle}
                description={longDescription}
            />
        );

        expect(screen.getByText(longTitle)).toHaveClass("break-words");
        expect(screen.getByText(longDescription)).toHaveClass("break-words");
    });
});
